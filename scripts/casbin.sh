#!/bin/bash

set -e
set -o pipefail

current_dir=`cd $(dirname $0) && pwd`

db_name=postgres
user=postgres
sql_file=${current_dir}/../init/casbin.sql
# for postgresql
psql -d ${db_name} -U ${user} -f ${sql_file}