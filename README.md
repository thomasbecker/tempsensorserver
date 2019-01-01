# tempsensorserver
Minimal Webserver using embedded jetty to serve temperature sensor data read from DS18B20 sensors on raspberry pi as json

Did use as few libraries as possible on purpose for the very simple task.

#### BUILD:
mvn clean install

#### INSTALL:
install jdk > 1.8 on raspberry (e.g. `sudo apt-get install openjdk-8-jdk`)
copy target/de.softwareschmied .homeintegrator.temp-sensor-server-1.0-SNAPSHOT.jar to raspberry (e.g. using scp)

#### Start
`java -jar TempSensorServer.jar`

Use screen or nohup to run it as a background daemon. Create some init scripts to keep it running after reboots of the raspberry.

E.g. `screen java -jar TempSensorServer.jar`

Example STDOUT of the server:
```
Jan 01, 2019 3:09:13 PM org.eclipse.jetty.util.log.Log initialized
INFO: Logging initialized @1046ms to org.eclipse.jetty.util.log.Slf4jLog
Jan 01, 2019 3:09:14 PM org.eclipse.jetty.server.Server doStart
INFO: jetty-9.4.z-SNAPSHOT; built: 2018-11-14T21:20:31.478Z; git: c4550056e785fb5665914545889f21dc136ad9e6; jvm 11.0.1+13-Raspbian-3
Jan 01, 2019 3:09:14 PM org.eclipse.jetty.server.AbstractConnector doStart
INFO: Started ServerConnector@b50428{HTTP/1.1,[http/1.1]}{0.0.0.0:8080}
Jan 01, 2019 3:09:14 PM org.eclipse.jetty.server.Server doStart
INFO: Started @2034ms
Jan 01, 2019 3:09:17 PM de.softwareschmied.homeintegrator.tempsensorserver.SensorServlet doGet
INFO: reading sensor data...
Jan 01, 2019 3:09:18 PM de.softwareschmied.homeintegrator.tempsensorserver.SensorServlet doGet
INFO: Returning sensor data: [{id: 0, value: 85.000},{id: 1, value: 21.562}]
```

#### Test
```
$ curl 192.168.188.40:8080/sensors
[{id: 0, value: 21.250},{id: 1, value: 21.312}]
```
