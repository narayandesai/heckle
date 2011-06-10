"""A simple web server that accepts POSTS containing a list of feed urls,
and returns the titles of those feeds.
"""
import datetime
import eventlet
import eventlet.semaphore
import json
import os
import logging
import sys
import socket
import time
from genshi.template import NewTextTemplate

#Create a new logging object for debugging statuments.
logging.basicConfig(level=logging.DEBUG)

# the pool provides a safety limit on our concurrency
#Eventlet is a free threading implementation for network information
pool = eventlet.GreenPool()

#Cannot find webpage
class PageLookupError(Exception):
    pass

#Cannot render the template for the image
class RenderError(Exception):
    pass

#Cannot set the values of the information for the build image
class AttributeResolutionError(Exception):
    pass

#Creates a dthandler object for datetime.datetime object
def dthandler(obj):
    if isinstance(obj, datetime.datetime):
        return obj.isoformat()
    return obj


#Creates the flunky master object. This object allows for the full
#management of the entire system. This server can wait indefinately
#for information to be sent to the sever. this server will do a look
#based on the node name and write it to a file. Thus effectively 
#taking all of the information from the flunkys and compiling it
#into a persistent list that is sent to the fctl. 
class fm(object):

#Creates an object known as root(currently a file on the local host)
    #This file structure should be the path to the server on the network.
    #Once this is created the root node has other variables appened to it
    #that allow it to be called. The static and dynamic names afford for 
    #static or dynamic allocation. The class also sets up a data dictionary
    #as wll as a eventlet.semaphore for concurrent networking operations. In 
    #essence it works the same as the threading paradigm for single non con-
    #current programming
    #A simple semaphore is called to wait for a thread. Information on the 
    #structure is at http://docs.python.org/release/2.5.2/lib/semaphore-objects.html. 
    def __init__(self, root):
        self.root = root
	self.flunkyIP = socket.gethostbyname(socket.getfqdn())
        self.static = root +'/repository/static'
        self.dynamic = root +'/repository/dynamic'
        self.data = dict()
        self.data_sem = eventlet.semaphore.Semaphore()
        logging.info("Starting")
        self.assert_setup('127.0.0.1', {'Image':'ubuntu-maverick-amd64'})


    #Sets up a variable that will hold the information for one build. Contained in the build
    #are various status variables that tell the program when somethng was allocated and the time
    #since the last activity. Maintains a count of errors and the number of times the function was
    #run. Contains in the class the addresses(hostnames) of all clients on the network for a request
    #DURING a BUILD.
    def assert_setup(self, address, info):
        newsetup = dict([('Allocated', datetime.datetime.now()), ('Counts', dict()), ('Errors', 0), 
                         ('Activity', datetime.datetime.now()), ('Info', list()), ('Status', 'Starting')])
        newsetup['Image'] = info['Image']
        if 'Extra' in info and info['Extra'] != None:
            newsetup['Extra'] = info['Extra']
        with self.data_sem:
            self.data[address] = newsetup

    #Creates a list of build variables for the script that will be rendered later on 
    #in the process. This will update the data dictionary with the new path to the 
    #address of the build script. Returns a data dictionary that contains the address
    #of the build environment, the path to it and updates the data in self.
    def build_vars(self, address, path):
        if address not in self.data:
            raise AttributeResolutionError
        data = dict([('Address', address), ('Path', path), 
                     ('Count', self.data[address]['Counts'].get(path, 0))]) 
        data.update(self.data[address])
        return data

    #Increments the count. A count is defined as when something occurs in the script. 
    #if an error occurs, set it to one. 
    def increment_count(self, address, path):
        with self.data_sem:
            try:
                self.data[address]['Counts'][path] += 1
            except:
                self.data[address]['Counts'][path] = 1

    #Find the data directory and load a static
    #build templeate if it exsists. There is no
    #dynamic template building here.
    def render_get_static(self, address, path):
        try:
            fname = self.static + '/' + path
            os.stat(fname)
            self.increment_count(address, path)
            return open(fname).read()
        except:
            raise RenderError

    #Tries to open a file to the dynamic file address on the server. 
    #checks to see if the file exsists and if not will raise and exeption. 
    #The script then creates a new class variable called build_vars
    #passing in the address and path of the script. This will then create 
    #a new template file increase the count and then return a template.
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

    #Creates a dictionary that is then returned to the caller that 
    #Contains information that is perteninet to the call. 
    def create_dict(self, address):
    	request = dict([('Address', address), ('Status', self.data[address]['Status']), ('Info', self.data[address]['Info'])])
	return request

    #def store(self, filename):
	#open(filename, 'w').write(json.dumps)

    def load(self, filename):
	dataFile = open(filename)
	self.data = json.load(dataFile)

    #used when the function is used. Will call for a start responce and then get
    #the messgae from the calling client. It will then process this message and 
    #then take that message and make decisions based on it. So far all that has been 
    #studied is the POST and /ctl    
    def __call__(self, environ, start_response):
        address = environ['REMOTE_ADDR']
	print 'address ' + address
        path = environ['PATH_INFO'][1:]
	
        #Drops into a request method conditional. So far have only seen values for
        #POST.
        if environ['REQUEST_METHOD'] == 'GET':

            if path == 'dump':
                start_response('200 OK', [('Content-type', 'application/json')])
                return json.dumps(self.data, default=dthandler)
            try:
                if path.startswith('static'):
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
                msg = json.loads(data)
                logging.info(msg['Address'] + " : " + msg['Message'])
                with self.data_sem:
                    self.data[msg['Address']]['Activity'] = datetime.datetime.now()
                    self.data[msg['Address']]['Info'].append(dict([('Time', time.time()), ('Message', msg['Message']), ('MsgType', 'Info')]))


            elif path == 'error':
                msg = json.loads(data)
                logging.error(msg['Address'] + " : " + msg['Message'])
                with self.data_sem:
                    self.data[msg['Address']]['Activity'] = datetime.datetime.now()
                    self.data[msg['Address']]['Errors'] += 1
                    self.data[msg['Address']]['Info'].append(dict([('Time', time.time()), ('Message', msg['Message']), ('MsgType', 'Error')]))
	            print 'aa', self.data[msg['Address']]['Info']


            elif path == 'ctl':
                msg = json.loads(data)
                logging.info("Allocating %s as %s" % (msg['Address'], msg['Image']))
                self.assert_setup(msg['Address'], msg)

	    #pass new structure in this area so that it can be read from fctl
	    #The message will have an address requested. The message will take
            #it and move it to him per request. 
	    #newsetup = dict([('Allocated', datetime.datetime.now()), ('Counts', dict()), ('Errors', 0), 
            #('Activity', datetime.datetime.now()), ('Info', list()), ('Status', 'Starting')])
            elif path == 'status':
                msg = json.loads(data)
                nodestatus = self.render_get_dynamic(msg['Address'], '../status').strip()
                self.data[address]['Status'] = nodestatus
                start_response('200 OK', [('Content-type', 'application/json')])
		created = self.create_dict(msg['Address'])

                return json.dumps(created, default=dthandler)


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
        repopath = os.getcwd()
    except:
        print "Usage: flunkymaster.py <repodir>"
        raise SystemExit, 1
    wsgi.server(eventlet.listen(('localhost', 8080)), fm(root=repopath))

