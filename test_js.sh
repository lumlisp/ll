#!/bin/bash

ll examples/test_js.ll > md.js
if [[ "$(node md.js)" == "<h1>header</h1>" ]]; then
	echo "ok"
else
	echo "fail"
fi
rm md.js
