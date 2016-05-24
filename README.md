
## Protocol buffers

protoc --go_out . snapshot_new.proto 

## Init scripts


```
pganalyze-collector --config=/etc/pganalyze-collector.conf -tv

sudo chkconfig --add pganalyze-collector
sudo chkconfig pganalyze-collector on
sudo service pganalyze-collector start
```


TODO:
- Encode /etc/pganalyze-collector.conf at build time (so you can run the collector command for testing purposes)


Resources
- https://www.nginx.com/resources/wiki/start/topics/examples/redhatnginxinit/#
- https://access.redhat.com/documentation/en-US/Red_Hat_Enterprise_Linux/6/html/Deployment_Guide/s2-services-chkconfig.html
- http://refspecs.linuxbase.org/LSB_3.1.0/LSB-Core-generic/LSB-Core-generic/iniscrptfunc.html
- http://refspecs.linuxfoundation.org/LSB_3.1.0/LSB-Core-generic/LSB-Core-generic/initsrcinstrm.html
- https://wiki.debian.org/LSBInitScripts
- https://wiki.debian.org/Debate/initsystem
- https://github.com/VividCortex/godaemon
