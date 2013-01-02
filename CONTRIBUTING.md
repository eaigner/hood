If you want to contribute, please follow these rules.

## Dialects

1. The dialect file name must reflect the actual dialect name, e.g. `sqlite3.go`
2. Unit and live test cases must be updated to run properly on the new dialect (see top of `dialects_test.go`)
3. Live tests should be commented out after testing and before committing, not to interfere with real unit tests.

## General

1. Pull requests containing commit messages with unappropriate content (e.g. smilies) will be rejected
2. Pull requests with pending `TODO(xyz):`'s will be rejected