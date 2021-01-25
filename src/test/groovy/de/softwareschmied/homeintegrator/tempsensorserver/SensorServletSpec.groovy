package de.softwareschmied.homeintegrator.tempsensorserver

import spock.lang.Specification

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
class SensorServletSpec extends Specification {
    SensorServlet sensorServlet = new SensorServlet()

    def "convert sensor set to json works"() {
        given:
        def sensors = [new Sensor("1", "22.444"), new Sensor("2", "23.222")] as Set
        when:
        def json = sensorServlet.getAsJson(sensors)
        then:
        json == "{\"sensors\": [{\"id\": \"1\", \"value\": \"22.444\"},{\"id\": \"2\", \"value\": \"23.222\"}]}"

    }
}
