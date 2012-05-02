#!/bin/sh

args=`getopt S:g:iex: $*`
errcode=$?
set -- $args

for i
do
    case "$i"
        in
        -S)
            serverurl="$2"
            shift ; shift;;
        -i)
            mode=info
            shift ;;
        -e)
            mode=err
            shift ;;
        -g)
            mode=get
            fpath="$2"
            shift ; shift ;;
        -x)
            mode=run
            fpath="$2"
            shift ; shift ;;
        --)
            shift ; break ;;
    esac
done

msg="$@"

if [ "$0" = "/opt/bootlocal.sh" ] ; then
    bootlocal=1
    for word in `cat /proc/cmdline` ; do
        var=`echo $word | awk -F= '{print $1}'`
        if [ "$var" = flunky ] ; then
            serverurl=`echo $word|awk -F= '{print $2}'`
        fi
    done
    mode=run
    fpath=install
fi

if [ -z "$serverurl" ] ; then
    echo "-S is required"
    exit 1
fi

if [ -z "$mode" ] ; then
    echo "one of -g, -x, -i, or -e is required"
    exit 1
fi

if [ "$mode" = "info" -o "$mode" = "error" ] ; then
    curl -d "{\"Message\": \"$msg\"}" "${serverurl}/$mode"
    exit $?
elif [ "$mode" = "get" ] ; then
    curl "${serverurl}/${fpath}"
    exit $?
elif [ "$mode" = "run" ] ; then
    tmpprefix=`basename $0`
    runpath=`mktemp /tmp/${tmpprefix}.XXXXXX` || exit 1
    while /bin/true; do 
      if [ ! -z "$bootlocal" ] ; then 
        curl -o "${runpath}" "${serverurl}/${fpath}" 2> /dev/null
      else
        curl -o "${runpath}" "${serverurl}/${fpath}" 
      fi

      rc="$?"
      if [ -z "$bootlocal" ] ; then
         break
      fi
      if [ "$rc" -eq 0 ] ; then
           break
      fi
      sleep 1
      if [ -z "$first" ] ; then 
        echo -n waiting for server
        first=1
      else
        echo -n .
      fi
    done 
    chmod +x "${runpath}"
    "${runpath}"
    rc=$?
    rm "${runpath}"
    exit "$rc"
fi
