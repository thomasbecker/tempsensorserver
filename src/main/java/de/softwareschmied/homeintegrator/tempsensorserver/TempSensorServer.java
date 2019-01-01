package de.softwareschmied.homeintegrator.tempsensorserver;

import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.servlet.ServletHandler;

/**
 * Created by Thomas Becker (thomas.becker00@gmail.com) on 2018-12-31.
 */
public class TempSensorServer {
    public static void main(String[] args) throws Exception {
        Server server = new Server(8080);
        ServletHandler handler = new ServletHandler();
        server.setHandler(handler);
        handler.addServletWithMapping(SensorServlet.class, "/sensors");
        server.start();
        server.join();
    }

}
