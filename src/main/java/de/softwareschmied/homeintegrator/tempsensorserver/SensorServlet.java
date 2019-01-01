package de.softwareschmied.homeintegrator.tempsensorserver;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.io.PrintWriter;
import java.util.Comparator;
import java.util.Set;
import java.util.stream.Collectors;

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
public class SensorServlet extends HttpServlet {
    private static final Logger LOG = LoggerFactory.getLogger(SensorServlet.class);

    private static SensorReader sensorReader = new SensorReader();

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response) {
        LOG.info("reading sensor data...");
        Set<Sensor> sensors = sensorReader.getSensors();
        String sensorData = getAsJson(sensors);
        response.setContentType("application/json");
        response.setStatus(HttpServletResponse.SC_OK);
        LOG.info("Returning sensor data: {}", sensorData);
        try (PrintWriter writer = response.getWriter()) {
            writer.println(sensorData);
        } catch (IOException e) {
            response.setStatus(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
            LOG.warn("Exception caught while writing response: {}", e.getMessage());
            LOG.info("Exception caught while writing response:", e);
        }
    }

    String getAsJson(Set<Sensor> sensors) {
        return String.format("[%s]", sensors.stream().sorted(Comparator.comparing(Sensor::getId)).map(this::getAsJson).collect(Collectors.joining(",")));
    }

    private String getAsJson(Sensor sensor) {
        return String.format("{id: %s, value: %s}", sensor.getId(), sensor.getValue());
    }
}
