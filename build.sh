#!/bin/bash

#################################################
# 打包相关，请勿随意更改
#################################################

params_num=$#

if [ $params_num -le 0 ]
then
    echo "未指定脚本参数【uat或者prod】"
    exit 255 
fi

PROJECT_NAME='boxfox_grpc_server'

function restorePath(){
    export PATH=$1
    if [ x"$2" != "x" ];then
        echo "goroot restore"
        export GOROOT=$2
        echo $GOROOT
    fi
    echo $PATH
}

mkdir -p src/$PROJECT_NAME
path=`pwd`
sysPath=$PATH
echo $sysPath
sysGroot=$GOROOT

echo $(go version)
sourcecode=`ls|grep -Ev 'src|pkg|bin|LAST_BUILD_USER'`
cp -r  $sourcecode src/$PROJECT_NAME
cd src/$PROJECT_NAME

env='prod'
if [ x"$1" != "x" ];then
  env=$1
fi

buildCmd="go build -o deploy/$env/bin/$PROJECT_NAME *.go"

if ls go.mod >/dev/null 2>&1; then
    `go mod edit -require=google.golang.org/grpc@v1.26.0`
    `go mod vendor`
else
	echo "not found go.mod file"
	exit 255
fi
echo $buildCmd
if ! $buildCmd ; then
  exit 255
fi

#编译后的上线打包目录创建
mkdir -p output/bin
mkdir -p output/conf
mkdir -p output/log
mkdir -p output/static

cp deploy/$env/bin/* output/bin
cp -f deploy/$env/conf/* output/conf
cp -r deploy/$env/shell/* output/
cp -r deploy/$env/data output/
#cp deploy/$env/perfcounter.json output/
cp -r static output/

#调整权限
chmod -R 755 output/
cp -r output $path/
rm -rf $path/src

