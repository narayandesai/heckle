"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import eventlet
import simplejson as json

# the pool provides a safety limit on our concurrency
pool = eventlet.GreenPool()

def app(environ, start_response):
    start_response('200 OK', [('Content-type', 'application/json')])
    # serve out some json
    return json.dumps({'StartOn': 'starton', 'Supplies': 'supplies', 'Requires': ['requires1', 'requires2'], 'Script': 'something'})

if __name__ == '__main__':
    from eventlet import wsgi
    wsgi.server(eventlet.listen(('localhost', 8080)), app)

