package de.softwareschmied.homeintegrator.tempsensorserver

import spock.lang.Specification

import java.nio.file.Paths

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
class SensorReaderSpec extends Specification {
    def static sensorReader = new SensorReader()

    def setupSpec(){
        sensorReader.sensorPath = this.class.getClassLoader().getResource("").file
    }

    def "get sensor returns values for two existing sensors"() {
        when:
        def sensors = sensorReader.sensors

        then:
        sensors.size() == 2
        sensors.each {
            assert it.id != null
            assert it.value == "25.625" || it.value == "27.525"
        }
    }
}
