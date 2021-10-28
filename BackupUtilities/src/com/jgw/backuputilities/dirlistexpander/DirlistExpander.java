package com.jgw.backuputilities.dirlistexpander;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.file.Files;
import java.nio.file.InvalidPathException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.LinkedList;
import java.util.List;
import java.util.stream.Collectors;

/** Create folders based on a list of directories from, e.g. 'dir /s /b'. */
public class DirlistExpander {

	public static void main(String[] args) throws IOException {
		createDirectories();
	}

	private static void createDirectories() throws IOException {

		Path root = Paths.get("cygdrive");

		Path dirlistFile = Paths.get("/home/jgw/dirlist.txt");

		BufferedReader br = new BufferedReader(new InputStreamReader((Files.newInputStream(dirlistFile))));

		while (true) {
			String line = br.readLine();
			if (line == null) {
				break;
			}

			if (!line.startsWith(root + "/")) {
				System.err.println("> " + line);
				continue;
			}

			try {

				Path path = root.relativize(Paths.get(line));

				Path newPath = Paths.get("/home/jgw/a", path.toString());
//			newPath.relativize(path);

				if (!newPath.toString().startsWith("c:\\delme\\a\\")) {
					System.out.println("err1:" + newPath);
					return;
				}

				if (newPath.toString().contains("\\..\\")) {
					System.out.println("err2: " + newPath);
					return;
				}

				Files.createDirectories(newPath);

			} catch (InvalidPathException ipe) {
				System.err.println("ipe, skipping: " + line);
			}

		}

	}

	private static void createDirectoryStructureBasedOnFolder() throws IOException {
		Path root = Paths.get("/home/jgw/Ephemeral/delme");

		List<Path> paths = new LinkedList<>();

		paths.add(root);

		while (paths.size() > 0) {

			Path removedPath = paths.remove(0);

			for (Path rpPath : Files.list(removedPath).collect(Collectors.toList())) {

				if (Files.isDirectory(rpPath)) {
					paths.add(rpPath);
				}

				Path relativePath = root.relativize(rpPath);
				System.out.println(relativePath);

				Files.createDirectories(Paths.get("/home/jgw/delme/b", relativePath.toString()));

			}

		}

	}

}
