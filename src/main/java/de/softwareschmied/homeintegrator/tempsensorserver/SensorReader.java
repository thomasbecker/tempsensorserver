package de.softwareschmied.homeintegrator.tempsensorserver;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.List;
import java.util.Scanner;
import java.util.Set;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import java.util.stream.Stream;

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
public class SensorReader {
    private Path sensorPath = Paths.get("/sys/devices/w1_bus_master1");

    public Set<Sensor> getSensors() {
        try (Stream<Path> paths = Files.walk(sensorPath)) {
            List<File> sensorFiles =
                    paths.filter(Files::isRegularFile).map(Path::toFile).filter(file -> file.getName().equals("w1_slave")).collect(Collectors.toList());
            return IntStream.range(0, sensorFiles.size()).mapToObj(i -> new Sensor(String.valueOf(i), getSensorValue(sensorFiles.get(i)))).collect(Collectors.toSet());
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

    /**
     * Override sensorPath for testing only
     */
    public void setSensorPath(String sensorPath) {
        this.sensorPath = Paths.get(sensorPath);
    }
}
