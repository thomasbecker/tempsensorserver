package de.softwareschmied.homeintegrator.tempsensorserver;

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
public class Sensor {
    private String id;
    private String value;

    public Sensor(String id, String value) {
        this.id = id;
        this.value = value;
    }

    public String getId() {
        return id;
    }

    public String getValue() {
        return value;
    }
}
