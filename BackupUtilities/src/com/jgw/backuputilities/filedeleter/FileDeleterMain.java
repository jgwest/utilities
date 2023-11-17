package com.jgw.backuputilities.filedeleter;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

public class FileDeleterMain {

	public static void main(String[] args) throws IOException {

		Path targetPath = Paths.get("E:\\delme\\restic-ouput");

		recurseFolder(targetPath);
	}

	private static void recurseFolder(Path folder) throws IOException {

		Files.list(folder).forEach(e -> {

			if (Files.isDirectory(e)) {
				try {
					recurseFolder(e);
				} catch (IOException e1) {
					throw new RuntimeException(e1);
				}
			} else {
				processFile(e);
			}

		});

	}

	private static void processFile(Path file) {

		try {
			if (file.toString().endsWith(".json")) {
				Files.delete(file);
			} else if (Files.size(file) > 1024 * 1024 * 0.6) {
				System.out.println(file);
				Files.delete(file);
			}
		} catch (IOException e) {
//			System.err.println(file);
			e.printStackTrace();
		}
	}

}
