#!/bin/bash
cd $(mktemp -d)
yum install epel-release -y
yum install python3 -y openssl
sleep 2
pip3 install --no-cache-dir /host && pip3 install pyinstaller
pyinstaller --onefile --name pgcdfga /host/pgcdfga_run.py 
cp dist/pgcdfga /host/pgcdfga.c7
