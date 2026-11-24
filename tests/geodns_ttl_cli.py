"""Real Terraform CLI acceptance using an isolated API contract simulator.

This supplements, rather than replaces, the separately opt-in real API tests.
No production API requests or credentials are used.
"""
import copy
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import subprocess
import tempfile
from threading import Thread, Lock
from urllib.parse import parse_qs, urlsplit
from urllib.request import Request, urlopen

import geodns_ttl_live as live

LOCATIONS = {'DEFAULT': '1', 'EUR': '5', 'BG': '30', 'US-1': '259'}


class API(BaseHTTPRequestHandler):
    rows = {str(i): {'id': str(i), 'host': 'existing-sibling', 'type': 'A',
                    'record': '192.0.2.' + str(i - 9000), 'ttl': '600',
                    'geodns-location': loc, 'status': 0, 'notes': 'existing'}
            for i, loc in ((9001, '5'), (9002, '30'))}
    baseline = copy.deepcopy(rows)
    sequence = 100
    lock = Lock()
    writes = []
    fail_endpoint = None

    def log_message(self, *args):
        pass

    def do_POST(self):
        with self.lock:
            self.handle_api()

    def handle_api(self):
        raw = self.rfile.read(int(self.headers['Content-Length']))
        p = json.loads(raw) if raw.lstrip().startswith(b'{') else {
            k: v[0] for k, v in parse_qs(raw.decode(), keep_blank_values=True).items()}
        endpoint = self.path.rsplit('/', 1)[-1].removesuffix('.json')
        try:
            assert str(p.get('sub-auth-id')) == '42' and p.get('auth-password') == 'synthetic-test-password'
            assert p.get('domain-name') == 'example.test'
            if endpoint in ('add-record', 'mod-record', 'delete-record'):
                self.writes.append((endpoint, str(p.get('record-id', ''))))
            assert endpoint != self.fail_endpoint, 'Simulated API failure'
            if endpoint == 'get-geodns-locations':
                result = [{'code': code, 'id': loc} for code, loc in LOCATIONS.items()]
            elif endpoint == 'get-available-ttl':
                result = ['300', '3600']
            elif endpoint == 'records':
                result = {k: v for k, v in self.rows.items()
                          if (not p.get('host') or v['host'] == p['host'])
                          and (not p.get('record-type') or v['type'] == p['record-type'])}
            elif endpoint in ('add-record', 'mod-record', 'delete-record'):
                key = str(p.get('record-id', ''))
                assert key not in self.baseline, 'Attempted mutation of existing record'
                if endpoint == 'delete-record':
                    assert key in self.rows
                    del self.rows[key]
                    result = {'status': 'Success'}
                else:
                    if endpoint == 'add-record':
                        assert 'terraform-geo-' in p['host']
                        type(self).sequence += 1
                        key = str(self.sequence)
                        row = {'id': key, 'type': p['record-type']}
                    else:
                        row = self.rows[key].copy()
                    for field in ('host', 'record', 'ttl'):
                        row[field] = str(p[field])
                    # ClouDNS resets an omitted GeoDNS selector to DEFAULT.
                    row['geodns-location'] = str(p.get('geodns-location') or LOCATIONS.get(p.get('geodns-code'), '1'))
                    for field in ('priority', 'weight', 'port') if row['type'] == 'SRV' else (
                            ('order', 'pref', 'flag', 'params', 'regexp', 'replace') if row['type'] == 'NAPTR' else ()):
                        row[field] = str(p.get(field, ''))
                    self.rows[key] = row
                    result = {'status': 'Success', 'data': {'id': int(key)}}
            else:
                raise AssertionError('Unexpected endpoint: ' + endpoint)
        except (AssertionError, KeyError) as error:
            result = {'status': 'Failed', 'statusDescription': str(error)}
        body = json.dumps(result).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def test_allow_overwrite():
    fixtures = {
        '9101': ('Terraform-Geo-Overwrite', '5'),
        '9102': ('terraform-geo-overwrite', '1'),
        '9103': ('terraform-geo-ambiguous', '1'),
        '9104': ('terraform-geo-ambiguous', '1'),
        '9105': ('terraform-geo-opt-out', '1'),
        '9106': ('', '1'),
    }
    for key, (host, location) in fixtures.items():
        API.rows[key] = {'id': key, 'host': host, 'type': 'A', 'record': '192.0.2.1',
                         'ttl': '300', 'geodns-location': location}
    initial = copy.deepcopy(API.rows)
    record = {'zone': 'example.test', 'type': 'A', 'value': '192.0.2.10', 'ttl': 3600}
    specs = {
        'adopted': dict(record, name='terraform-geo-overwrite', geodnscode='EUR', allow_overwrite=True),
        'apex': dict(record, name='@', allow_overwrite=True),
        'created': dict(record, name='terraform-geo-create-if-missing', allow_overwrite=True),
        'opt_out': dict(record, name='terraform-geo-opt-out'),
    }
    with tempfile.TemporaryDirectory(prefix='cloudns-overwrite-') as directory:
        work = Path(directory)
        cli_config = work / 'dev.tfrc'
        provider_dir = str(Path(os.environ['CLOUDNS_TF_DEV_PROVIDER_DIR']).resolve())
        cli_config.write_text('provider_installation {\n dev_overrides {\n'
                              ' "ClouDNS/cloudns" = ' + json.dumps(provider_dir) + '\n }\n}\n')
        env = dict(os.environ, TF_CLI_CONFIG_FILE=str(cli_config), TF_INPUT='0', CHECKPOINT_DISABLE='1')

        def write_config(records):
            (work / 'main.tf.json').write_text(json.dumps({
                'terraform': {'required_providers': {'cloudns': {'source': 'ClouDNS/cloudns'}}},
                'provider': {'cloudns': {}}, 'resource': {'cloudns_dns_record': records},
            }))

        def terraform(*args, expected=0):
            result = subprocess.run([os.environ.get('TERRAFORM_BINARY', 'terraform'), *args],
                                    cwd=work, env=env, capture_output=True, text=True, timeout=60)
            assert result.returncode == expected, result.stdout + result.stderr
            return result

        write_config(specs)
        terraform('apply', '-auto-approve', '-parallelism=1', '-no-color')
        state = json.loads(terraform('show', '-json').stdout)
        resources = state['values']['root_module']['resources']
        ids = {resource['name']: resource['values']['id'] for resource in resources}
        assert ids['adopted'] == '9101' and ids['apex'] == '9106', ids
        assert ids['created'] not in initial and ids['opt_out'] not in initial, ids
        for name in ('adopted', 'apex'):
            row = API.rows[ids[name]]
            assert row['record'] == '192.0.2.10' and row['ttl'] == '3600', row
        assert API.rows['9101']['geodns-location'] == '5'
        assert API.rows['9102'] == initial['9102'] and API.rows['9105'] == initial['9105']
        terraform('plan', '-detailed-exitcode', '-no-color')

        writes = len(API.writes)
        specs['adopted'].pop('allow_overwrite')
        write_config(specs)
        terraform('apply', '-auto-approve', '-no-color')
        assert len(API.writes) == writes, API.writes[writes:]
        terraform('plan', '-detailed-exitcode', '-no-color')

        terraform('destroy', '-auto-approve', '-no-color')
        assert not set(ids.values()) & API.rows.keys()
        unchanged = {key: row for key, row in initial.items() if key not in ids.values()}
        assert API.rows == unchanged

        writes = len(API.writes)
        write_config({'ambiguous': dict(record, name='terraform-geo-ambiguous', allow_overwrite=True)})
        failed = terraform('apply', '-auto-approve', '-no-color', expected=1)
        assert 'multiple A DNS records match' in failed.stderr
        assert len(API.writes) == writes

        write_config({'status': dict(record, name='terraform-geo-opt-out', status=0, allow_overwrite=True)})
        failed = terraform('apply', '-auto-approve', '-no-color', expected=1)
        assert 'cannot set status' in failed.stderr
        assert len(API.writes) == writes

        write_config({'existing': dict(record, name='terraform-geo-opt-out', allow_overwrite=True)})
        try:
            for endpoint, attempts in (('records', []), ('mod-record', [('mod-record', '9105')])):
                API.fail_endpoint = endpoint
                failed = terraform('apply', '-auto-approve', '-no-color', expected=1)
                assert 'Simulated API failure' in failed.stderr
                assert API.writes[writes:] == attempts
                assert API.rows == unchanged
        finally:
            API.fail_endpoint = None
        print('Overwrite: adoption, create fallback, opt-out, no-op plan, and API failure guards verified', flush=True)


def main():
    server = ThreadingHTTPServer(('127.0.0.1', 0), API)
    endpoint = 'http://127.0.0.1:' + str(server.server_port)
    os.environ.update(CLOUDNS_GEO_TTL_TEST='1', CLOUDNS_GEO_TEST_DOMAIN='example.test',
                      CLOUDNS_SUB_AUTH_ID='42', CLOUDNS_PASSWORD='synthetic-test-password',
                      CLOUDNS_TEST_API_ENDPOINT=endpoint)
    os.environ.pop('CLOUDNS_AUTH_ID', None)
    def redirected(request, **kwargs):
        assert urlsplit(request.full_url).netloc == 'api.cloudns.net'
        return urlopen(Request(endpoint + urlsplit(request.full_url).path,
                               data=request.data), **kwargs)
    live.urlopen = redirected
    Thread(target=server.serve_forever, daemon=True).start()
    try:
        live.main()
        assert API.rows == API.baseline
        test_allow_overwrite()
    finally:
        server.shutdown()
        server.server_close()


if __name__ == '__main__':
    main()
