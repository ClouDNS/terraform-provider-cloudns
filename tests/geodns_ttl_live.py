"""Opt-in Terraform CLI GeoDNS TTL checks; only UUID test records are writable.

Build the provider, then set CLOUDNS_GEO_TTL_TEST=1, CLOUDNS_GEO_TEST_DOMAIN,
CLOUDNS_SUB_AUTH_ID, CLOUDNS_PASSWORD, CLOUDNS_TF_DEV_PROVIDER_DIR and optionally
TERRAFORM_BINARY. No existing records are imported, modified or deleted.
"""

import json
import os
from pathlib import Path
import subprocess
import tempfile
from urllib.parse import urlencode
from urllib.request import Request, urlopen
from uuid import uuid4


def main():
    assert os.environ.get('CLOUDNS_GEO_TTL_TEST') == '1', 'Explicit opt-in required'
    domain = os.environ['CLOUDNS_GEO_TEST_DOMAIN']
    auth = {'sub-auth-id': os.environ['CLOUDNS_SUB_AUTH_ID'],
            'auth-password': os.environ['CLOUDNS_PASSWORD']}

    def api(endpoint, **fields):
        payload = dict(auth, **{'domain-name': domain}, **fields)
        request = Request('https://api.cloudns.net/dns/{}.json'.format(endpoint),
                          data=urlencode(payload).encode())
        with urlopen(request, timeout=60) as response:
            result = json.load(response)
        if isinstance(result, dict) and result.get('status') == 'Failed':
            raise RuntimeError(result.get('statusDescription', 'API error'))
        return result

    def rows():
        return api('records', **{'include-notes': 1})

    def unchanged(actual, expected):
        changed = [key for key, row in expected.items() if actual.get(key) != row]
        assert not changed, 'Original row IDs changed: {}'.format(changed)

    locations = {row['code']: row['id'] for row in api('get-geodns-locations')}
    assert {'DEFAULT', 'EUR', 'BG', 'US-1'} <= set(locations)
    assert {300, 3600} <= set(map(int, api('get-available-ttl')))
    baseline = rows()
    prefix = 'terraform-geo-' + uuid4().hex
    assert not any(prefix in row['host'] for row in baseline.values())
    specs = {}
    for index, record_type in enumerate(('A', 'AAAA', 'ALIAS', 'CNAME', 'SRV', 'NAPTR')):
        attrs = {'zone': domain, 'name': prefix + '-' + record_type.lower(),
                 'type': record_type, 'ttl': 300, 'geodnscode': 'BG',
                 'value': 'target.example.org'}
        if record_type == 'A':
            attrs['value'] = '192.0.2.1'
        elif record_type == 'AAAA':
            attrs['value'] = '2001:db8::1'
        elif record_type == 'SRV':
            attrs.update(name='_probe._tcp.' + attrs['name'], priority=10, weight=20, port=443)
        elif record_type == 'NAPTR':
            attrs.update(value='', order='10', pref='20', flag='S', params='SIP+D2U',
                         regexp='', replace='target.example.org')
        specs[record_type.lower()] = attrs
    for index, code in enumerate(('DEFAULT', 'EUR', 'US-1')):
        specs['a_' + str(index)] = dict(specs['a'], name=prefix + '-a',
                                       value='192.0.2.{}'.format(index + 2), geodnscode=code)
    specs['imported'] = dict(specs['a'], name=prefix + '-imported', value='192.0.2.99')
    specs['imported'].pop('geodnscode')
    names = {attrs['name'] for attrs in specs.values()}
    imported_id = None

    with tempfile.TemporaryDirectory(prefix='cloudns-geo-ttl-') as directory:
        work = Path(directory)
        config = work / 'main.tf.json'
        cli_config = work / 'dev.tfrc'
        provider_dir = str(Path(os.environ['CLOUDNS_TF_DEV_PROVIDER_DIR']).resolve())
        cli_config.write_text('provider_installation {\n dev_overrides {\n'
                              ' "ClouDNS/cloudns" = ' + json.dumps(provider_dir) + '\n }\n}\n')
        env = dict(os.environ, TF_CLI_CONFIG_FILE=str(cli_config), TF_INPUT='0',
                   CHECKPOINT_DISABLE='1')
        env.pop('TF_LOG', None)
        env.pop('TF_LOG_PATH', None)
        env.pop('CLOUDNS_AUTH_ID', None)

        def write_config():
            config.write_text(json.dumps({
                'terraform': {'required_providers': {'cloudns': {'source': 'ClouDNS/cloudns'}}},
                'provider': {'cloudns': {}},
                'resource': {'cloudns_dns_record': specs}}, indent=2))

        def terraform(*args, expected=(0,)):
            result = subprocess.run([os.environ.get('TERRAFORM_BINARY', 'terraform'),
                                     *args], cwd=work, env=env, capture_output=True, text=True)
            if result.returncode not in expected:
                output = result.stdout + result.stderr
                for secret in auth.values():
                    output = output.replace(secret, '***')
                raise RuntimeError(output)
            return result

        def state():
            return json.loads(terraform('show', '-json').stdout)['values']['root_module']['resources']

        try:
            result = api('add-record', **{'record-type': 'A', 'host': specs['imported']['name'],
                                         'record': '192.0.2.99', 'ttl': 300, 'geodns-code': 'BG'})
            imported_id = str(result['data']['id'])
            assert imported_id not in baseline
            write_config()
            terraform('import', '-no-color', 'cloudns_dns_record.imported', domain + '/' + imported_id)
            terraform('apply', '-auto-approve', '-parallelism=1', '-no-color')
            resources = state()
            assert len(resources) == 10
            owned = {resource['values']['id'] for resource in resources}
            assert not owned & set(baseline)
            before = rows()
            unchanged(before, baseline)
            assert set(before) == set(baseline) | owned
            for resource in resources:
                attrs = resource['values']
                assert attrs['geodnslocation'] == before[attrs['id']]['geodns-location']
            print('Create/import: 10 owned records; location captured in state', flush=True)

            for attrs in specs.values():
                attrs['ttl'] = 3600
            write_config()
            terraform('plan', '-out=ttl.plan', '-parallelism=1', '-no-color')
            plan = json.loads(terraform('show', '-json', 'ttl.plan').stdout)
            updates = [change for change in plan['resource_changes']
                       if change['change']['actions'] != ['no-op']]
            assert len(updates) == 10
            for update in updates:
                change = update['change']
                assert change['actions'] == ['update']
                assert {key for key in change['before']
                        if change['before'][key] != change['after'].get(key)} == {'ttl'}
            assert rows() == before, 'Plan sent writes'
            terraform('apply', '-parallelism=1', '-no-color', 'ttl.plan')
            after = rows()
            unchanged(after, baseline)
            assert set(after) == set(before)
            for key in owned:
                assert str(after[key]['ttl']) == '3600'
                assert {k: v for k, v in after[key].items() if k != 'ttl'} == {
                    k: v for k, v in before[key].items() if k != 'ttl'}
            terraform('plan', '-detailed-exitcode', '-parallelism=1', '-no-color')
            print('TTL-only plan/apply/plan: all 10 IDs and non-TTL API fields preserved', flush=True)

            # Verify an intentional code change is not overridden by the old computed ID.
            a_id = next(r['values']['id'] for r in state() if r['name'] == 'a')
            specs['a']['geodnscode'] = 'EUR'
            write_config()
            terraform('apply', '-auto-approve', '-parallelism=1', '-no-color')
            moved = rows()
            unchanged(moved, baseline)
            assert moved[a_id]['geodns-location'] == locations['EUR']
            assert moved[a_id]['record'] == after[a_id]['record']
            terraform('plan', '-detailed-exitcode', '-parallelism=1', '-no-color')
            print('Explicit GeoDNS code change also converged with the same ID', flush=True)
        finally:
            # Reconcile ambiguous create outcomes using UUID names, not broad zone deletes.
            current = rows()
            cleanup = {key for key, row in current.items() if row['host'] in names}
            if imported_id is not None and imported_id in current:
                cleanup.add(imported_id)
            assert not cleanup & set(baseline)
            for key in cleanup:
                api('delete-record', **{'record-id': key})
            final = rows()
            unchanged(final, baseline)
            assert set(final) == set(baseline)
            print('Cleanup: {} original records unchanged; zero temporary records'.format(len(baseline)), flush=True)


if __name__ == '__main__':
    main()
