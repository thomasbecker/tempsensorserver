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
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.ReentrantLock;
import java.util.stream.Collectors;

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
public class SensorServlet extends HttpServlet {
    private static final Logger LOG = LoggerFactory.getLogger(SensorServlet.class);

    private static final SensorReader sensorReader = new SensorReader();

    private static final ReentrantLock lock = new ReentrantLock();

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response) {
        try {
            boolean lockAcquired = lock.tryLock(30, TimeUnit.SECONDS);
            if (lockAcquired) {
                try {
                    LOG.info("reading sensor data...");
                    Set<Sensor> sensors = sensorReader.getSensors();
                    String sensorData = getAsJson(sensors);
                    response.setContentType("application/json");
                    response.setStatus(HttpServletResponse.SC_OK);
                    LOG.info("Returning sensor data: {}", sensorData);
                    writeSensorData(response, sensorData);
                } finally {
                    lock.unlock();
                }
            } else {
                LOG.info("Couldn't acquire lock. Returning empty response body.");
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOG.info("Interrupted: ", e);
        }
    }

    private void writeSensorData(HttpServletResponse response, String sensorData) {
        try (PrintWriter writer = response.getWriter()) {
            writer.println(sensorData);
        } catch (IOException e) {
            response.setStatus(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
            LOG.warn("Exception caught while writing response: {}", e.getMessage());
            LOG.info("Exception caught while writing response:", e);
        }
    }

    String getAsJson(Set<Sensor> sensors) {
        return String.format("{\"sensors\": [%s]}",
                sensors.stream().sorted(Comparator.comparing(Sensor::getId)).map(this::getAsJson).collect(Collectors.joining(",")));
    }

    private String getAsJson(Sensor sensor) {
        return String.format("{\"id\": \"%s\", \"value\": \"%s\"}", sensor.getId(), sensor.getValue());

    }
}
