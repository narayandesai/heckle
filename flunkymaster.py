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

class LookupError(Exception):
    pass

class RenderError(Exception):
    pass

class AttributeResolutionError(Exception):
    pass

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

    def build_vars(self, address, path):
        if address not in self.data:
            raise AttributeResolutionError
        return  dict([('address', client), ('path', path), ('pagecount', self.data[address]['counts'].get(path))]) \
        + self.data[address]

    def render_get_static(self, address, path):
            try:
                fname = self.static + '/' + path
                os.stat(fname)
                if path not in self.data[address]['counts']:
                    self.data[address]['counts'][path] = 0
                self.data[address]['counts'][path] += 1
                return open(fname).read()
            except:
                raise RenderError

    def render_get_dynamic(self, address, path):
        try:
            vars = self.build_vars(self.address, path)
        except AttributeResolutionError:
            raise RenderError

    def __call__(self, environ, start_response):
        address = environ['REMOTE_ADDR']
        path = environ['PATH_INFO'][1:]
        if environ['REQUEST_METHOD'] == 'GET':
            if path == 'status':
                start_response('200 OK', [('Content-type', 'application/json')])
                return json.dumps(self.data, default=dthandler)
            try:
                if path.startswith('static/'):
                    data = self.render_get_static(address, path[path.find('/'):])
                    start_response('200 OK', [('Content-type', 'application/binary')])
                    return data
                elif path.startswith('dynamic'):
                    data = self.render_get_dynamic(address, path[path.find('/'):])
                    start_response('200 OK', [('Content-type', 'application/binary')])
                    return data
                else:
                    raise LookupError
            except LookupError:
                start_response('404 Not Found', [('Content-Type', 'text/plain')])
                return ['Not Found\r\n']
            except RenderError:
                start_response('500 Server Error', [('Content-Type', 'text/plain')])
                return ''
            except:
                log.exception("Get failure")
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

