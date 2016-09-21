Packaging of the pganalyze collector
====================================

The scripts and Makefile in this directory take care of building packages of the
pganalyze collector for all recent Linux-based operating systems.

The process is split into three parts, and three corresponding directories:

1. **src/**: Build an appropriate deb/rpm package for the required init systems (systemd, upstart, sysvinit)
2. **test/**: Test the deb/rpm package installation on all supported distributions
3. **repo/**: Sign the deb/rpm packages using the signing key (https://keybase.io/pganalyze) and
   synchronize the S3 hosted package repositories for each distribution

All packages are 64-bit only.

Contributions welcome!


Supported Linux distributions
-----------------------------

* RHEL/CentOS
  * 6
  * 7
* Fedora
  * 24
* Amazon Linux (2014.03 or newer)
* Debian
  * Jessie
  * Stretch (Testing)
* Ubuntu
  * Precise (12.04 LTS)
  * Trusty (14.04 LTS)
  * Xenial (16.04 LTS)

These are the distributions that are automatically tested and for which repositories exist - others might work as well.


Minimum required glibc version
------------------------------

When changing the initial package build step one needs to be careful to not increase the minimum required glibc version accidentally.

Currently the following minimum glibc versions apply:

* RPM, sysvinit: glibc 2.12 (CentOS 6)
* RPM, systemd: glibc 2.17 (CentOS 7)
* DEB, upstart: glibc 2.12 (Ubuntu Precise / 12.04 LTS)
* DEB, systemd: glibc 2.19 (Debian Jessie)


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
