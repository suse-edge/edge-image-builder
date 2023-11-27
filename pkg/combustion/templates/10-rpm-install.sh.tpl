#!/bin/bash
set -euo pipefail

#  Template Fields
#  RPMs - A string that contains all of the RPMs present in the user created config directory, separated by spaces.

rpm -ivh --nosignature {{.RPMs}}