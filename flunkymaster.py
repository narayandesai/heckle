"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import eventlet
import json

# the pool provides a safety limit on our concurrency
pool = eventlet.GreenPool()

def app(environ, start_response):
    start_response('200 OK', [('Content-type', 'text/plain')])
    return "#!/bin/sh\n\necho foo\necho bar\n"

if __name__ == '__main__':
    from eventlet import wsgi
    wsgi.server(eventlet.listen(('localhost', 8080)), app)

