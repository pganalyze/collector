Packaging of the pganalyze collector
====================================

The scripts and Makefile in this directory take care of building packages of the
pganalyze collector for all recent Linux-based operating systems.

The process is split into three parts, and three corresponding directories:

1. **src/**: Build an appropriate deb/rpm package for the systemd init system
2. **test/**: Test the deb/rpm package installation on all supported distributions
3. **repo/**: Sign the deb/rpm packages using the signing key (https://keybase.io/pganalyze) and
   synchronize the S3 hosted package repositories for each distribution

All packages are built for both 64-bit X86 (amd64) and 64-bit ARMv8 (arm64/aarch64) targets.

Contributions welcome!


Supported Linux distributions
-----------------------------

See packages/test/Makefile file.

These are the distributions that are automatically tested and for which repositories exist - others might work as well.


Minimum required glibc version
------------------------------

When changing the initial package build step one needs to be careful to not increase the minimum required glibc version accidentally.

Currently the following minimum glibc versions apply:

* RPM, systemd: glibc 2.28 (built on Rocky Linux 8, oldest supported target is RHEL 8 with glibc 2.28)
* DEB, systemd: glibc 2.34 (built on Debian Bookworm, oldest supported target is Ubuntu Jammy with glibc 2.35)

Note that the minimum version is determined by the glibc symbol versions referenced by the
built binaries, not by the glibc version of the build image. The build verifies this using
`packages/src/check_glibc.sh`, which fails the build if a binary requires a newer glibc.


Requirements
------------

Docker is needed for all package building and repo sync needs. In addition you'll also need
the keybase.io client, as well as the [AWS CLI](https://aws.amazon.com/cli/).


License
-------

The packaging and init scripts for pganalyze-collector are licensed under the 3-clause BSD license,
see LICENSE file in the root directory for details.

Alternatively, you may also use and copy these packaging scripts under the CC0 license (https://creativecommons.org/publicdomain/zero/1.0/).

Packaging can be painful and everybody wins by reusing existing logic.
