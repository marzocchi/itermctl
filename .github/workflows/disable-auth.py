#!/usr/bin/env python3
import os
import sys

filename = os.path.expanduser(sys.argv[1])
magic = "61DF88DC-3423-4823-B725-22570E01C027"
contents = filename.encode("utf-8").hex() + " " + magic

with open(filename, mode='w') as f:
    f.write(contents)
    f.flush()
