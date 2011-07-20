"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import eventlet
import eventlet.semaphore
import json
import os
import logging
import time
import datetime
import socket
from genshi.template import NewTextTemplate

''' the pool provides a safety limit on our concurrency
Eventlet is a free threading implementation for network information'''
pool = eventlet.GreenPool()

'''Cannot find webpage'''
class PageLookupError(Exception):
    pass

'''Cannot render the template for the image'''
class RenderError(Exception):
    pass

'''Cannot set the values of the information for the build image'''
class AttributeResolutionError(Exception):
    pass

class ImageResolutionError(Exception):
    pass

'''Creates the flunky master object. This object allows for the full
management of the entire system. This server can wait indefinately
for information to be sent to the sever. this server will do a look
based on the node name and write it to a file. Thus effectively 
taking all of the information from the flunkys and compiling it
into a persistent list that is sent to the fctl.''' 
class fm(object):

    '''Creates an object known as root(currently a file on the local host)
    This file structure should be the path to the server on the network.
    Once this is created the root node has other variables appened to it
    that allow it to be called. The static and dynamic names afford for 
    static or dynamic allocation. The class also sets up a data dictionary
    as wll as a eventlet.semaphore for concurrent networking operations. In 
    essence it works the same as the threading paradigm for single non con-
    current programming
    A simple semaphore is called to wait for a thread. Information on the 
    structure is at http://docs.python.org/release/2.5.2/lib/semaphore-objects.html.''' 

    def __init__(self, root):
        self.root = root
        self.static = os.path.join(root, 'static')
        self.dynamic = os.path.join(root, 'dynamic')
        self.datafile = os.path.join(root, 'data.json')
        staticdatapath = os.path.join(root, 'staticVars.json')
        self.data = dict()
        self.data_sem = eventlet.semaphore.Semaphore()
        msgFormat = "%(asctime)s - %(levelname)s - %(message)s"
        logfile = os.path.join(root, "flunky.log")
        try:
            os.stat(logfile)
        except:
            open(logfile, 'w')
        logging.basicConfig(filename=logfile, level=logging.DEBUG, format=msgFormat)

        try:
            self.static_build = json.load(open(staticdatapath))
        except: 
            logging.error("Failed to load static build variables %s " %(staticdatapath))
            self.static_build = dict()
        self.load()
        logging.info("Starting")
        self.assert_setup('127.0.0.1', {'Image':'ubuntu-maverick-amd64'})
	
    def load(self):
        try:
            self.data = json.load(open(self.datafile))
        except:
            logging.error("Failed to load datafile %s" % self.datafile)
            self.data = dict()

    def store(self):
        json.dump(self.data, open(self.datafile, 'w'))

    '''Sets up a variable that will hold the information for one build. Contained in the build
    are various status variables that tell the program when somethng was allocated and the time
    since the last activity. Maintains a count of errors and the number of times the function was
    run. Contains in the class the addresses(hostnames) of all clients on the network for a request
    DURING a BUILD.'''
    def assert_setup(self, address, info):
        imageDir = self.root + '/images/' + info['Image'] 
        try:
            os.stat(imageDir)
        except:
            logging.error('Failed to find requested image %s' % (imageDir))
            raise ImageResolutionError

        newsetup = dict([('Allocated', long(time.mktime(time.localtime()))), ('Counts', dict()), ('Errors', 0), 
                         ('Activity', long(time.mktime(time.localtime()))), ('Info', list())])
        newsetup['Image'] = info['Image']
        newsetup['Counts']['bootconfig'] = 0
        if 'Extra' in info and info['Extra'] != None:
            newsetup['Extra'] = info['Extra']
        with self.data_sem:
            self.data[address] = newsetup
            self.store()

    '''Creates a list of build variables for the script that will be rendered later on 
    in the process. This will update the data dictionary with the new path to the 
    address of the build script. Returns a data dictionary that contains the address
    of the build environment, the path to it and updates the data in self.'''
    def build_vars(self, address, path):
        if address not in self.data:
            raise AttributeResolutionError
        data = dict()
        # handle static settings first, so dynamic values supercede them
        data.update(self.static_build)
        dynamic = dict([('Address', address), ('Path', path),('Count', self.data[address]['Counts'].get(path, 0)), ('Counts' , self.data[address]['Counts'])])
        dynamic['IMAGE'] = self.data[address]['Image']
        data.update(dynamic)
        return data

    '''Increments the count. A count is defined as when something occurs in the script. 
    if an error occurs, set it to one. '''
    def increment_count(self, address, path):
        with self.data_sem:
            try:
                self.data[address]['Counts'][path] += 1
            except:
                self.data[address]['Counts'][path] = 1
            self.store()

    '''Find the data directory and load a static
    build templeate if it exsists. There is no
    dynamic template building here.'''
    def render_get_static(self, address, path):
        try:
            fname = self.static + '/' + path
            os.stat(fname)
            self.increment_count(address, path)
            return open(fname).read()
        except:
            raise RenderError

    '''Tries to open a file to the dynamic file address on the server. 
    checks to see if the file exsists and if not will raise and exeption. 
    The script then creates a new class variable called build_vars
    passing in the address and path of the script. This will then create 
    a new template file increase the count and then return a template.'''
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

    ''''Tries to find the path for the image that is requested. 
    returns the path of that image if it exsists or an 
    empty string otherwise'''
    def render_image_path(self, address, toRender):
        imageName = self.data[address]['Image']
        requestData = self.root + '/images/' + imageName + '/' + toRender
        try:
            os.stat(requestData)
        except:
            logging.error('Failed to find template %s' % (requestData))
            raise PageLookupError
        try:
            bvars = self.build_vars(address, toRender)
        except AttributeResolutionError:
            raise RenderError
        with open(requestData) as infile:
            tmpl = NewTextTemplate(infile.read())
        self.increment_count(address, toRender)
        try: 
            return tmpl.generate(**bvars).render('text')
        except:
            logging.exception("Genshi template Error")
            raise RenderError

    '''used when the function is used. Will call for a start responce and then get
    the messgae from the calling client. It will then process this message and 
    then take that message and make decisions based on it. So far all that has been 
    studied is the POST and /ctl'''    
    def __call__(self, environ, start_response):
        address = environ['REMOTE_ADDR']
        path = environ['PATH_INFO'][1:]
       
        if environ['REQUEST_METHOD'] == 'GET':
            
            if path == 'dump':
                start_response('200 OK', [('Content-type', 'application/json')])
                return json.dumps(self.data)
            try:
                if path.startswith('static'):
                    data = self.render_get_static(address, path[path.find('/'):])
                    start_response('200 OK', [('Content-type', 'application/binary')])
                    return data


                elif path.startswith('dynamic'):
                    data = self.render_get_dynamic(address, path[path.find('/'):])
                    start_response('200 OK', [('Content-type', 'application/binary')])
                    return data

                elif path in ['bootconfig', 'install']:
                    template = self.render_image_path(address, path)
                    start_response('200 OK', [('Content-type', 'application/binary')])
                    return template

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
            msg = json.loads(data)

            if path == 'info':
                logging.info(address + " : " + msg['Message'])
                with self.data_sem:
                    self.data[address]['Activity'] = time.mktime(time.localtime())
                    self.data[address]['Info'].append(dict([('Time', long(time.mktime(time.localtime()))), 
                                                            ('Message', msg['Message']), ('MsgType', 'Info')]))
                    self.store()


            elif path == 'error':
                logging.error(address + " : " + msg['Message'])
                with self.data_sem:
                    self.data[address]['Activity'] = time.mktime(time.localtime())
                    self.data[address]['Errors'] += 1
                    self.data[address]['Info'].append(dict([('Time', long(time.mktime(time.localtime()))), 
                                                            ('Message', msg['Message']), ('MsgType', 'Error')]))
                    self.store()


            elif path == 'ctl':
                for client in msg['Addresses']:
                    logging.info("Allocating %s as %s" % (socket.gethostbyname(client), msg['Image']))
                    try:
                        self.assert_setup(socket.gethostbyname(client), msg)
                    except:
                        start_response('500 Server Error', [('Content-Type', 'text/plain')])
                        return 'Cannot find image file'

            elif path == 'status':
                ret = dict()
                for client in msg['Addresses']:
                    try:
                        cstatus = self.render_image_path(socket.gethostbyname(client), 'status').strip()
                    except:
                        start_response('500 Server Error', [('Content-Type', 'text/plain')])
                        return 'Image not found'
                    with self.data_sem:
                        ret[client] = dict([('Status', cstatus), ('LastActivity', long(self.data[socket.gethostbyname(client)]['Activity']))])
                        ret[client]['Info'] = [imsg for imsg in self.data[socket.gethostbyname(client)]['Info'] if imsg['Time'] > msg['Time']] 
                        
                start_response('200 OK', [('Content-type', 'application/json')])
                return json.dumps(ret)


            else:
                start_response('404 Not Found', [('Content-Type', 'text/plain')])
                return ''
            start_response('200 OK', [('Content-type', 'application/octet-stream')])
            return ""
        start_response('404 Not Found', [('Content-Type', 'text/plain')])
        return ['Not Found\r\n']


if __name__ == '__main__':
    from eventlet import wsgi
    try:
        repopath = os.getcwd()+ '/repository'
    except:
        print "Usage: flunkymaster.py <repodir>"
        raise SystemExit, 1
    wsgi.server(eventlet.listen(('localhost', 8080)), fm(root=repopath))

