package org.acme;

import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

@Path("/env")
public class GreetingResource {

	@GET
	@Produces(MediaType.TEXT_PLAIN)
	public String hello() {

		String envVars = System.getenv().entrySet().stream()
				.sorted((a, b) -> a.getKey().toLowerCase().compareTo(b.getKey().toLowerCase())).map(e -> {
					return e.getKey() + ": " + e.getValue() + "\n";
				}).reduce((a, b) -> (a + b)).get();

		return String.format("Environment Variables:\n\n%s", envVars);
	}
}