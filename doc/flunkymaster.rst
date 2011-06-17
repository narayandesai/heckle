---Introduction ---
Using the flunkymaster system will allow all users of the system to image new machines dynamically without the hassle of trying to configure the nodes idependently. Flunkymaster will provide a faster and safer environment to build nodes on any cluster that runs the daemon. The images are stored on the nodes main server and new images can be pushed up and created almost instantly by followng the specified guidelines. The flunkymaster system allows for dynamic templating. Dynamic templates allow any user of the system to set configuration variables in a configuration file to control the build environment. This procedure allows for greater control of the build envoironment as well as a method to create and distrubute new images with little hassle.  

---Required files ---
When creating new image for the server it is imperative that the following data and path directories exsist so that the flunkymaster system can configure images correctly. 
    - At the root of the directory of flunky you will need the executables for the program that shall not be touched and the repository directory.  
      - All modifications need to be done using configuration files and dynamic templates. 
    - In the repository directory there will be various files available:
      - A data.json that is created as a backup of the system data
      - A flunky.log that will log all system events of the flunkymaster service
      - A staticVars.json file that will allow the user to configure a customized build
    - You will need 3 directories under the repository directory
     - static
     - dynamic
     - images
       - Each directory under the images directory will contain a folder with the image name that is trying to be configured. 
       - Each directory must contain two files
         - A status file to show the progress of the build
         - bootconfig file to load the correct build environment under GPXE
         - You can optionally include an install script for the build
   

---Visual Representation---
Contained here is a list of the directory and directory contents for the sytem:

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

---Reasons for the files to exist---
Here is the reason that certain files are needed in the repository folder. 
    The data.json file is created when the server starts for the first time. If this file is not deleted it will be reloaded when the flunkymaster server restarts. This file holds the state information about the server as well any activity that was happening when the server was stopped. This is as persisted json file and should not be changed.
    The flunky.log is also created on server startup and is simply a log file that maintains a record of info and error messages sent to the server. 
    The staticVars.json file will allow the user of the system to input various static build variables into build scripts in order to create a dynamic build script. 
    The images directory needs to exist and share the name of the image that is being rendered. This is a non-negotiable issue. If the wrong image name is supplied, the image will not be rendered correctly. 
    The bootconfig file will allow the program to load in the ram disk the correct boot image with GPXE in order to image the disks.  
    The status file is used internally to report states of build back through the system. It is not wise to change this information unless you really know what you are doing. Since this information is used internally by the system it is read only. 
    Optionally you can include an install script that can be dynamically build an image using the Genshi template system.

---The Genshi template system---
The flunkymaster system is written entirely in python and uses the Genshi template system to dynamically create templates. Genshi is like many other python templating systems. The Genshi system allows for the information in the build script to be rendered dynamically as requested. This will allow for a faster creation to deployment time. If more information about the Genshi system is required or anything Genshi related please feel free to visit:  http://genshi.edgewall.org/

---How to create a dynamic template---
By using the Genshi template system, image and node builds are easy to implement and easy to configure. Genshi will take a build script that is rendered statically by the user of the system and change all variables in the script preceded with the $. This method is better explained with the following example:

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
    we can partake in eating various fruits. I look forward to seeing you again. 

Regards, 
Jane

The static variables file should also include just the name of the variable to be rendered without the preceding $.


***A word of caution***
Most shell scripts and various other languages use the $ to denote variable reference. The Genshi template system uses the exact same reference methods. In order to work around this particular problem, in the shell scripts or any script that is written for the system please precede any variables that need to be left unchanged for the system with $$ instead of $. This will alleviate any errors that can be caused by the Genshi system. Additionally if you have a variable that needs to have the $ escape character left in the script the user needs to escape the escape character with $$$ instead of $$. 


---Not yet implemented---
  -Dynamic build script through functions
  -Dynamic and static files in static and dynamic folders 

