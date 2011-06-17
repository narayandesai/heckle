---Introduction ---
Using the new Heckle system will allow all users of the system to image new machines on the fly without the hassle of waiting for the node to allocate or trying to configure the nodes themselves. The new Heckle will provide a faster and safer environemnt ot build nodes on any cluster that runs the daemon. The images themselves are stored on the nodes main server and many other new images can be pushed up and created almost instantly. The new Heckle system allows for dynamic templates. Dynamic templates allow any user of the system to set configureation variables in a configuration file. What is gained by this is better error reporting as well a greater control of the build. 

---Required files ---
When creating new image for the server it is imperative that the following information be available so that the new Heckle system can configure images correctly. 
   - You will need 3 directories under the repository directory
     - Static
     - Dynamic
     - images
       - Each directory under the images directory will contain a folder with the image name that is trying to be configured. 
       - Each directory must contain two files
         - A status file to show the progress of the build
         - Bootconfig file to load the correct build environemtn under GPXE
         - You can optionally include an install script for the build
    - In the repository directory there will be various files available to you
      - A data.json that is created as a backup of the system should it go down
      - A flunky.log that will log all system events for later viewing
      - A staticVars.json file that will allow the user to configure the build in a customized way
    -At the root of the dictory of flunky you will need the executables for the program that shall not be touched.  
     - All modifications need to be done using configuration files and dynamic templates. 

---Visual Representation---
A graphic will better facilitate what is needed for the build:

flunky(folder)
  \
   \
    fctl
    flunky
    flunkymaster.py
    repository (folder)
       \                          \                            \
        \                          \                            \
         data.json
         flunky.log
         staticVars.json
         dynamic(folder)               static(folder)           images(folder)
           \                             \                        \
            \                             \                        \
            test                           foo                     imagename(folder)
                                                                      \
                                                                       \
                                                                        bootconfig
                                                                        status
                                                                        install(optional)

---Reasons for the files to exsist---
Here is the reason that certain files are needed in the repository folder. 
    The data.json file is created when the server starts for the first time. If this file is not deleted it will be reloaded when the server starts back up. This file holds the state information about the    server as well as the clients currently connected to the server. This is as persisted json file and should not be changed.
    The flunky.log is also created on server startup and is simpily a logfile that is written to when the system sends info and error messages. 
    The staticVars.json file will allow the user of the system to imput various static build variables into such implementations as the build scripts in order to create a dynamic and more customized build script. 
    The images directory needs to exsist and also needs to have the exact name of the image that you are trying to build. This is a non-negotibile issue. If you do not supply the right image name the image will not build. The images directory also needs to have a bootconfig file and a status file. 
    The bootconfig file will allow the program to load in the ram disk the correct boot image with GPXE in order to get the image to start building the disks. This will allow for the behind the scenes work for a build so to speak. 
    The status file is used internally to report states of the machine back through the system. It is not wise to change this information unless you really know what you are doing. Since this information is used internallly by the system it is read only. 
    Optionally you can include an install script that can be build dynamically by the program using the Genshi templating system. The install script can also be dynamically built in order to create a new build script. *Not yet implemented. The install script can fill out the fields that were specified in the static build script if nessesary. 

---The Genshi templating sytem---
The flunkymaster system is written entirely in python and uses the Genshi templating system to dynamically create templates. Genshi is like many other templating systems out the for python. The Genshi system allows for the information in the build script to be rendered dynamically without needed to create a separtate build script for each machine. This will allow for a faster time from creation to deployment of the system that needs to be created on the node server. If you would like more information about the Genshi system or anything Genshi related please feel free to visit:  http://genshi.edgewall.org/

---How to create a dynamic template---
Most people that would like to implement a dynamic image script would like it to be as easy as possible. Genshi is the tool that allows this to happen. Genshi will take a build script that is rendered statically  by the user of the system and change all variables in the script preceeded with the $. A simple example to better show this is as follows:

Hello $name, 

    Going to the $activity with you last weekend was an exciting adventure. I hope that we can do it again in the
    future. Please feel free to visit any time you like. My address is $address. Maybe the next time we venture out
    we can get some {$food}s. I look forward to seeing you again. 

Regards, 

$closing

If we set name to John, activity to football game, address to 123 Any Street, food to fruit and closing to Jane in our staticVar.json file this will be the output: 

Hello John, 

    Going to the football game with you last weekend was an exciting adventure. I hope that we can do it again in the
    future. Please feel free to visit any time you like. My address is 123 Any Street. Maybe the next time we venture out
    we can get some fruits. I look forward to seeing you again. 

Regards, 

Jane


***A word of caution***
Most shell scripts use the $ to denote variable reference. The Genshi templating system does exactaly the same. In order to work around this particular problem, in the shell scripts or any script that is written for the system please preceede any variables that need to be left unchanged for the system with $$ instead of $. This will allieviate any errors that can be caused by the Genshi system. Additionally if you have a variable that needs to have the $ escape character left in the script the user needs to escape the escape character with $$$ instead of $$. 


---Not yet implemented---
  -Dynamic build script through functions
  -Dynamic and static files in static and dynamic folders 

