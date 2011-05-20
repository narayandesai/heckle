"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import datetime
import eventlet
import json
import os
import logging

logging.basicConfig(level=logging.DEBUG)

# the pool provides a safety limit on our concurrency
pool = eventlet.GreenPool()

def dthandler(obj):
    if isinstance(obj, datetime.datetime):
        return obj.isoformat()
    return obj

class fm(object):
    def __init__(self, root):
        self.root = root
        self.static = root +'/static'
        self.data = dict()
        logging.info("Starting")
        self.data['127.0.0.1'] = dict([('image', 'ubuntu-maverick-amd64'), ('allocated', datetime.datetime.now()), ('activity', datetime.datetime.now()), ('counts', dict()), ('errors', 0)])

    def __call__(self, environ, start_response):
        address = environ['REMOTE_ADDR']
        path = environ['PATH_INFO'][1:]
        if environ['REQUEST_METHOD'] == 'GET':
            if path == 'status':
                start_response('200 OK', [('Content-type', 'application/json')])
                return json.dumps(self.data, default=dthandler)
            try:
                fname = self.static + '/' + path
                os.stat(fname)
                start_response('200 OK', [('Content-type', 'application/octet-stream')])
                return open(fname).read()
            except:
                start_response('404 Not Found', [('Content-Type', 'text/plain')])
                return ['Not Found\r\n']
        elif environ['REQUEST_METHOD'] == 'POST':
            data = environ['wsgi.input'].read()
            if path == 'info':
                logger.info(address + "INFO" +  data)
                self.data[address]['activity'] = datetime.datetime.now()
            elif path == 'error':
                logger.error(address + data)
                self.data[address]['activity'] = datetime.datetime.now()
                self.data[address]['errors'] += 1
            else:
                start_response('404 Not Found', [('Content-Type', 'text/plain')])
                return ''
            start_response('200 OK', [('Content-type', 'application/octet-stream')])
            return ""
        start_response('404 Not Found', [('Content-Type', 'text/plain')])
        return ['Not Found\r\n']


if __name__ == '__main__':
    from eventlet import wsgi
    wsgi.server(eventlet.listen(('localhost', 8080)), fm(root='/Users/desai/tmp/flunky'))

