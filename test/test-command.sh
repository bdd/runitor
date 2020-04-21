#!/usr/bin/env bash

echo stdout: foo
echo stderr: bar >&2
echo stdout: baz

exit 33
