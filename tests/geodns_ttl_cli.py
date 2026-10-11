"""Real Terraform CLI acceptance using an isolated API contract simulator.

This supplements, rather than replaces, the separately opt-in real API tests.
No production API requests or credentials are used.
"""
import copy
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
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
    finally:
        server.shutdown()
        server.server_close()


if __name__ == '__main__':
    main()
