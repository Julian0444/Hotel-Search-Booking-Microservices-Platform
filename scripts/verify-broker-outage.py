#!/usr/bin/env python3
"""Acceptance against an isolated Compose stack; real APIs, broker and Mongo.

COMPOSE_PROJECT_NAME=hotel-closure python3 scripts/verify-broker-outage.py
Stops/restarts only this project's RabbitMQ/hotels-api. Keeps audit reservations;
never clears databases. Do not run concurrently with E2E or a live demo.
"""
import datetime as dt
import json
import os
from pathlib import Path
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

ROOT = Path(__file__).resolve().parents[1]
os.chdir(ROOT)
project = os.environ.get('COMPOSE_PROJECT_NAME', '')
if not project.startswith('hotel-closure'):
    raise SystemExit('Use an isolated COMPOSE_PROJECT_NAME beginning hotel-closure; original volumes are not test fixtures.')
BASE = os.environ.get('TEST_API_BASE', 'http://localhost:5173/api/v1')
config = {}
for line in (ROOT / '.env').read_text().splitlines():
    if '=' in line and not line.lstrip().startswith('#'):
        key, value = line.split('=', 1)
        config[key.strip()] = value.strip()


def request(method, path, data=None, token='', key='', expected=None):
    headers = {'Accept': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    if key:
        headers['Idempotency-Key'] = key
    if data is not None:
        headers['Content-Type'] = 'application/json'
    req = urllib.request.Request(BASE + path, data=None if data is None else json.dumps(data).encode(), headers=headers, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=25)
    except urllib.error.HTTPError as error:
        response = error
    body = response.read()
    payload = json.loads(body) if body else None
    if expected is not None and response.code != expected:
        raise AssertionError(f'{method} {path}: expected {expected}, got {response.code}: {payload}')
    return response.code, payload, response.headers


def compose(*args):
    return subprocess.run(['docker', 'compose', *args], check=True, text=True, capture_output=True).stdout


def eventually(check, timeout=90):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if check():
            return
        time.sleep(1)
    raise AssertionError('condition did not converge within deadline')


admin = request('POST', '/login', {'username': config['ADMIN_USERNAME'], 'password': config['ADMIN_PASSWORD']}, expected=200)[1]['data']['token']
customer = request('POST', '/login', {'username': 'demo', 'password': 'DemoCliente123'}, expected=200)[1]['data']['token']
label = 'Outage' + uuid.uuid4().hex[:10]
base_hotel = dict(name=label, address='Test 1', city='Córdoba', country='Argentina', description='', state='', phone='', email='', price_per_night=10, rating=0, available_rooms=1, check_in_time='14:00', check_out_time='10:00', amenities=[], images=[])


def create(suffix):
    hotel = {**base_hotel, 'name': label + ' ' + suffix}
    hotel_id = request('POST', '/admin/hotels', hotel, token=admin, expected=201)[1]['data']['id']
    return hotel_id, hotel


booking_id, _ = create('Booking')
edit_id, edit_hotel = create('Edit')
delete_id, _ = create('Delete')
search_path = '/search?' + urllib.parse.urlencode({'q': label, 'limit': 100})

def search_rows():
    return request('GET', search_path, expected=200)[1]['data']

eventually(lambda: len(search_rows()) == 3)
check_in = (dt.date.today() + dt.timedelta(days=30)).isoformat()
check_out = (dt.date.today() + dt.timedelta(days=32)).isoformat()
booking = {'hotel_id': booking_id, 'check_in': check_in, 'check_out': check_out, 'num_rooms': 1, 'num_guests': 1}
key = str(uuid.uuid4())
measurements = {}
try:
    compose('stop', 'rabbitmq')
    # Poll dependency signal rather than assuming that stopping a container was observed.
    def broker_down():
        health = json.loads(compose('exec', '-T', 'hotels-api', 'wget', '-qO-', 'http://127.0.0.1:8081/readyz'))
        return health['checks']['rabbitmq'] == 'down' and health['checks']['mongo'] == 'ok'
    eventually(broker_down, 20)
    start = time.monotonic()
    reservation = request('POST', '/reservations', booking, token=customer, key=key, expected=201)[1]['data']['id']
    measurements['book_ms_broker_down'] = round((time.monotonic() - start) * 1000, 1)
    available = {'hotel_ids': [booking_id], 'check_in': check_in, 'check_out': check_out}
    assert request('POST', '/hotels/availability', available, expected=200)[1]['data'][booking_id] is False
    replay = request('POST', '/reservations', booking, token=customer, key=key, expected=201)
    assert replay[1]['data']['id'] == reservation and replay[2]['Idempotency-Replayed'] == 'true'
    request('POST', '/reservations', {**booking, 'num_guests': 2}, token=customer, key=key, expected=409)

    compose('restart', 'hotels-api')
    eventually(lambda: request('GET', '/hotels/' + booking_id)[0] == 200, 30)
    assert request('POST', '/reservations', booking, token=customer, key=key, expected=201)[1]['data']['id'] == reservation
    start = time.monotonic()
    request('DELETE', '/reservations/' + reservation, token=customer, expected=204)
    measurements['cancel_ms_broker_down'] = round((time.monotonic() - start) * 1000, 1)
    request('DELETE', '/reservations/' + reservation, token=customer, expected=204)
    assert request('POST', '/hotels/availability', available, expected=200)[1]['data'][booking_id] is True
    assert all(value < 5000 for value in measurements.values()), measurements

    created_id, _ = create('CreatedWhileOffline')
    edited = {**edit_hotel, 'rating': 0, 'available_rooms': 0, 'price_per_night': 0, 'city': 'Mendoza', 'amenities': [], 'images': []}
    request('PUT', '/admin/hotels/' + edit_id, edited, token=admin, expected=200)
    persisted = request('GET', '/hotels/' + edit_id, expected=200)[1]['data']
    assert persisted['city'] == 'Mendoza' and persisted['available_rooms'] == 0 and persisted['images'] == []
    request('DELETE', '/admin/hotels/' + delete_id, token=admin, expected=204)
    request('GET', '/hotels/' + delete_id, expected=404)
    request('GET', '/hotels/' + created_id, expected=200)

    # Read durable state directly, independent of HTTP projections.
    mongo_check = '''
const uri='mongodb://root:'+encodeURIComponent(process.env.MONGO_INITDB_ROOT_PASSWORD)+'@127.0.0.1:27017/admin?directConnection=true';
const d=new Mongo(uri).getDB('hotels-api');
const ids=JSON.parse(process.env.VERIFY_IDS);
print(JSON.stringify({reservations:d.reservations.countDocuments({hotel_id:ids.booking}),
keys:d.idempotency_keys.countDocuments({reservation_id:ids.reservation}),
booked:d.reservation_inventory.find({hotel_id:ids.booking}).toArray().map(x=>x.booked),
created:d.hotels.countDocuments({_id:ObjectId(ids.created)}),
deleted:d.hotels.countDocuments({_id:ObjectId(ids.deleted)}),
capacity:d.hotels.findOne({_id:ObjectId(ids.edited)}).available_rooms}));
'''
    ids = json.dumps(dict(booking=booking_id, reservation=reservation, created=created_id, deleted=delete_id, edited=edit_id))
    durable = json.loads(compose('exec', '-T', '-e', 'VERIFY_IDS=' + ids, 'mongo', 'mongosh', '--nodb', '--quiet', '--eval', mongo_check))
    assert durable == dict(reservations=1, keys=1, booked=[0, 0], created=1, deleted=0, capacity=0), durable

    # Same URI as before, no manual reindex. Periodic reconciliation must heal
    # all three publish-after-write gaps even while RabbitMQ remains offline.
    def reconciled():
        rows = {h['id']: h for h in search_rows()}
        return delete_id not in rows and created_id in rows and rows.get(edit_id, {}).get('city') == 'Mendoza'
    eventually(reconciled, 90)
finally:
    compose('start', 'rabbitmq')


def broker_recovered():
    health = json.loads(compose('exec', '-T', 'hotels-api', 'wget', '-qO-', 'http://127.0.0.1:8081/readyz'))
    return health['checks']['rabbitmq'] == 'ok'
eventually(broker_recovered, 60)
request('PUT', '/admin/hotels/' + edit_id, {**edited, 'city': 'Salta'}, token=admin, expected=200)
eventually(lambda: any(h['id'] == edit_id and h['city'] == 'Salta' for h in search_rows()), 30)
# Remove only the two reservation-free resources created by this run.
request('DELETE', '/admin/hotels/' + edit_id, token=admin, expected=204)
request('DELETE', '/admin/hotels/' + created_id, token=admin, expected=204)
print(json.dumps({'status': 'passed', 'project': project, 'durable': durable, 'latency_measurements': measurements,
                  'retained_audit_hotel': booking_id, 'retained_cancelled_reservation': reservation}, indent=2))
