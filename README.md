<div align="center">

# P4Go - A P4API derived API

[![Docs](https://img.shields.io/badge/Docs-Official-blue.svg)](https://help.perforce.com/helix-core/apis/p4go/current/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.txt)
[![Support](https://img.shields.io/badge/Support-Official-green.svg)](mailto:support@perforce.com)

### P4Go is a wrapper for the P4 C++ API in Go.

P4Go is a Go module that provides an API to P4 Server. Using P4Go is faster than using the command-line interface in scripts, because multiple command can be executed on a single connection, and because it returns P4 Server responses as Go structs.

</div>


## Requirements

Go 1.24

The P4API relevant to your platform:
* Linux x86_64 - http://ftp.perforce.com/perforce/r25.2/bin.linux26x86_64/p4api-glibc2.12-openssl3.tgz
* Linux aarch64 - http://ftp.perforce.com/perforce/r25.2/bin.linux26aarch64/p4api-openssl3.tgz
* MacOS - https://ftp.perforce.com/perforce/r25.2/bin.macosx12u/p4api-openssl3.tgz
* Windows - https://ftp.perforce.com/perforce/r25.2/bin.mingw64x64/p4api-openssl3_gcc8_win32_seh.zip
* Windows - https://ftp.perforce.com/perforce/r25.2/bin.mingw64x64/p4api-openssl3_gcc8_win32_sjlj.zip
* Windows - https://ftp.perforce.com/perforce/r25.2/bin.mingw64x64/p4api-openssl3_gcc8_posix_seh.zip
* Windows - https://ftp.perforce.com/perforce/r25.2/bin.mingw64x64/p4api-openssl3_gcc8_posix_sjlj.zip

OpenSSL 3
* On Linux, this is typically the libssl-dev package
* On MacOS/Windows you'll need prebuilt platform specific binaries

## Build flags

* Linux
```sh
go env -w CGO_CPPFLAGS="-I<absolute path to Perforce C++ API>/include -g"
go env -w CGO_LDFLAGS="-L<absolute path to Perforce C++ API>/lib -lp4api -lssl -lcrypto"
```

* MacOS
```sh
go env -w CGO_CPPFLAGS="-I<absolute path to Perforce C++ API>/include -g"
go env -w CGO_LDFLAGS="-L<absolute path to Perforce C++ API>/lib -L<absolute path to OpenSSL libraries matching Perforce C++ API> -lp4api -lssl -lcrypto -framework ApplicationServices -framework Foundation -framework Security"
```

* Windows
```sh
go env -w CGO_ENABLED=1
go env -w CGO_CPPFLAGS="-I<absolute path to Perforce C++ API>/include -DOS_NT -g"
go env -w CGO_LDFLAGS="-L<absolute path to Perforce C++ API>/lib -L<absolute path to OpenSSL libraries matching Perforce C++ API> -lp4api -lssl -lcrypto -lcrypt32 -lws2_32  -lole32 -lshell32 -luser32 -ladvapi32 -lole32 -pthread -v"
```

## Documentation

Official documentation is located on the [Perforce website](https://help.perforce.com/helix-core/apis/p4go/current/)

## Support

P4Go is officially supported by Perforce.
Pull requests will be managed by Perforce's engineering teams. We will do our best to acknowledge these in a timely manner based on available capacity.  
Issues will not be managed on GitHub. All issues should be recorded via [Perforce's standard support process](https://www.perforce.com/support/request-support).

## License
This project is licensed under the MIT License. See [LICENSE](LICENSE.txt) for details.

### Third-Party Notices
This project includes third-party components. Their licenses and attributions are listed in [THIRD-PARTY-NOTICES](THIRD-PARTY-NOTICES.txt).