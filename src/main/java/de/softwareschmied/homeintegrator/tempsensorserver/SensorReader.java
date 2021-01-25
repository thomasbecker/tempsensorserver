package de.softwareschmied.homeintegrator.tempsensorserver;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.*;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import java.util.stream.Stream;

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
public class SensorReader {
    private static final Logger logger = LoggerFactory.getLogger(SensorReader.class);

    private Path sensorPath = Paths.get("/sys/devices/w1_bus_master1");

    public Set<Sensor> getSensors() {
        try (Stream<Path> paths = Files.walk(sensorPath)) {
            List<File> sensorFiles =
                    paths.filter(Files::isRegularFile).map(Path::toFile).filter(file -> file.getName().equals("w1_slave")).collect(Collectors.toList());
            Set<Sensor> sensors = IntStream.range(0, sensorFiles.size()).mapToObj(i -> new Sensor(String.valueOf(i), getSensorValue(sensorFiles.get(i)))).collect(Collectors.toSet());
            sensors.addAll(getDht22Values());
            return sensors;
        } catch (IOException e) {
            throw new IllegalStateException(e);
        }
    }

    private String getSensorValue(File file) {
        try (Scanner scanner = new Scanner(file)) {
            String value = scanner.findAll("[^t]+t=(\\d+)$").findFirst().get().group(1);
            StringBuilder sb = new StringBuilder(value);
            sb.insert(2, ".");
            return sb.toString();
        } catch (FileNotFoundException e) {
            throw new IllegalStateException(e);
        }
    }

    private Set<Sensor> getDht22Values() {
        ProcessBuilder builder = new ProcessBuilder();
        builder.command("/usr/bin/python3", "/home/pirate/dht_out.py");
        builder.directory(new File(System.getProperty("user.home")));
        builder.redirectErrorStream(true);
        int exitCode;
        try {
            logger.info("Executing python script");
            Process process = builder.start();
            exitCode = process.waitFor();
            logger.info("python script exited with: {}", exitCode);
            String output =
                    new BufferedReader(new InputStreamReader(process.getInputStream())).lines()
                            .findFirst().orElse("0 0");
            assert exitCode == 0;
            String[] split = output.split(" ");
            Sensor temp = new Sensor("100", split[0]);
            Sensor humidity = new Sensor("101", split[1]);
            return Set.of(temp, humidity);
        } catch (IOException e) {
            logger.warn("Error reading dht value: ", e);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            logger.warn("Error reading dht value: ", e);
        }
        return Collections.emptySet();
    }

    /**
     * Override sensorPath for testing only
     */
    public void setSensorPath(String sensorPath) {
        this.sensorPath = Paths.get(sensorPath);
    }
}
