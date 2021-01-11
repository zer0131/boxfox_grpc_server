
#!/bin/bash

source ~/.bash_profile

#module config
MODULE_NAME='boxfox_grpc_server'

function start() {
    sleep 1
    bin/gosuv -c conf/config.yml start-server
}

function stop() {
    bin/gosuv -c conf/config.yml shutdown
}

function reload() {
    PID=$(cat .gosuv.pid)
    kill -USR1 $PID

}

if [ $1 == 'start' ]; then
    start
fi

if [ $1 == 'stop' ]; then
    stop
fi

if [[ $1 = 'reload' ]]; then
    reload
fi
