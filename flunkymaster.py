"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import eventlet
import json
import os

# the pool provides a safety limit on our concurrency
pool = eventlet.GreenPool()

class fm(object):
    def __init__(self, static):
        self.static = static

    def __call__(self, environ, start_response):
        if environ['REQUEST_METHOD'] == 'GET':
            try:
                fname = self.static + '/' + environ['PATH_INFO'][1:]
                os.stat(fname)
                start_response('200 OK', [('Content-type', 'application/octet-stream')])
                return open(fname).read()
            except:
                start_response('404 Not Found', [('Content-Type', 'text/plain')])
                return ['Not Found\r\n']
        elif environ['REQUEST_METHOD'] == 'POST':
            print environ
            data = environ['wsgi.input'].read()
            print environ['REMOTE_ADDR'], data
            start_response('200 OK', [('Content-type', 'application/octet-stream')])
            return ""
        start_response('404 Not Found', [('Content-Type', 'text/plain')])
        return ['Not Found\r\n']


if __name__ == '__main__':
    from eventlet import wsgi
    wsgi.server(eventlet.listen(('localhost', 8080)), fm(static='/Users/desai/tmp/flunky/static'))

