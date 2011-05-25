"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import datetime
import eventlet
import eventlet.semaphore
import json
import os
import logging
from genshi.template import NewTextTemplate

logging.basicConfig(level=logging.DEBUG)

# the pool provides a safety limit on our concurrency
pool = eventlet.GreenPool()

class PageLookupError(Exception):
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
        self.dynamic = root +'/dynamic'
        self.data = dict()
        self.data_sem = eventlet.semaphore.Semaphore()
        logging.info("Starting")
        self.assert_setup('127.0.0.1', {'Image':'ubuntu-maverick-amd64'})

    def assert_setup(self, address, info):
        newsetup = dict([('Allocated', datetime.datetime.now()), ('Counts', dict()), ('Errors', 0), 
                         ('Activity', datetime.datetime.now())])
        newsetup['Image'] = info['Image']
        if 'Extra' in info and info['Extra'] != None:
            newsetup['Extra'] = info['Extra']
        with self.data_sem:
            self.data[address] = newsetup

    def build_vars(self, address, path):
        if address not in self.data:
            raise AttributeResolutionError
        data = dict([('Address', address), ('Path', path), 
                     ('Count', self.data[address]['Counts'].get(path, 0))]) 
        data.update(self.data[address])
        return data

    def increment_count(self, address, path):
        with self.data_sem:
            try:
                self.data[address]['Counts'][path] += 1
            except:
                self.data[address]['Counts'][path] = 1

    def render_get_static(self, address, path):
        try:
            fname = self.static + '/' + path
            os.stat(fname)
            self.increment_count(address, path)
            return open(fname).read()
        except:
            raise RenderError

    def render_get_dynamic(self, address, path):
        fname = self.dynamic + '/' + path
        try:
            os.stat(fname)
        except:
            raise PageLookupError
        try:
            bvars = self.build_vars(address, path)
        except AttributeResolutionError:
            raise RenderError

        # grab the requested template
        with open(fname) as infile:
            tmpl = NewTextTemplate(infile.read())

        # increment access count
        self.increment_count(address, path)
        # sick genshi on that template and return it
        try:
            return tmpl.generate(**bvars).render('text')
        except:
            logging.exception("Genshi template error")
            raise RenderError

    def __call__(self, environ, start_response):
        address = environ['REMOTE_ADDR']
        path = environ['PATH_INFO'][1:]
        if environ['REQUEST_METHOD'] == 'GET':
            if path == 'dump':
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
                    raise PageLookupError
            except PageLookupError:
                start_response('404 Not Found', [('Content-Type', 'text/plain')])
                return ['Not Found\r\n']
            except RenderError:
                start_response('500 Server Error', [('Content-Type', 'text/plain')])
                return ''
            except:
                logging.exception("Get failure")
        elif environ['REQUEST_METHOD'] == 'POST':
            data = environ['wsgi.input'].read()
            if path == 'info':
                logging.info(address + " INFO: " +  data)
                with self.data_sem:
                    self.data[address]['Activity'] = datetime.datetime.now()
            elif path == 'error':
                logging.error(address + " ERROR: " + data)
                with self.data_sem:
                    self.data[address]['Activity'] = datetime.datetime.now()
                    self.data[address]['Errors'] += 1
            elif path == 'ctl':
                msg = json.loads(data)
                logging.info("Allocating %s as %s" % (msg['Address'], msg['Image']))
                self.assert_setup(msg['Address'], msg)
            elif path == 'status':
                msg = json.loads(data)
                data = self.render_get_dynamic(msg['Address'], '../status')
                print ":", data, ":"
                start_response('200 OK', [('Content-type', 'application/octet-stream')])
                return data
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

