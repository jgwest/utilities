package com.jgw;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.stream.Collectors;

import com.jgw.mt.TMFWritableDatabase;
import com.jgw.util.Util;

public class Main {

	private static final long MAX_SIZE = 1024 * 1024 * 1024;
	private static final long MIN_SIZE = 32 * 1024;

	private static long filesProcessed = 0;

	public static void main(String[] args) throws IOException {

		Path targetPath = Path.of("c:\\");

		Path rootPath = Path.of("c:\\delme\\file-database");

		processDirectory(targetPath, rootPath);

		// hash file
		// acquire lock on sha zip
		// read sha zip
		// write sha zip

	}

	private static void processDirectory(Path targetDir, Path rootDir) throws IOException {

		for (Path p : Util.listFilesInPath(targetDir).stream().collect(Collectors.toList())) {

			if (Files.isDirectory(p)) {

				processDirectory(p, rootDir);

			} else {
				processFile(p, rootDir);
			}

		}

	}

	private static void processFile(Path path, Path rootPath) throws IOException {

		filesProcessed++;

		if (filesProcessed % 100 == 0) {
			System.out.println(filesProcessed);
		}

		long fileSize = Files.size(path);

		if (fileSize > MAX_SIZE && fileSize < MIN_SIZE) {
			return;
		}

		InputStream is = Files.newInputStream(path);
		String shaString = Util.getSHA256(is);
		is.close();

		Path destPath = Util.generateOutputPath(shaString, rootPath);
		Files.createDirectories(destPath.getParent());

		String output = "";
		if (Files.exists(destPath)) {
			output = Util.readSingleEntryFromZIPFileAsString(destPath);
		}

		output += shaString + " " + fileSize + " " + path.toString() + "\n";

		TMFWritableDatabase.writeToFile(output, destPath);

	}

}
