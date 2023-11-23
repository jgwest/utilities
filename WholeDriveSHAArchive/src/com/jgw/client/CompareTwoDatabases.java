package com.jgw.client;

import java.io.BufferedOutputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

import com.jgw.mt.DirectoryReadableDatabase;
import com.jgw.mt.IReadableDatabase;
import com.jgw.util.Util;

public class CompareTwoDatabases {

	public static void main(String[] args) throws IOException {

		IReadableDatabase otherDatabase = new DirectoryReadableDatabase(Path.of("C:\\db-everything"));

		Path remoteDB = Path.of("C:\\db-everything-else");

		try (OutputStream mismatchLogOutputStream = new FileOutputStream(Paths.get("c:\\mismatch-log.txt").toFile());
				BufferedOutputStream bufferedOS = new BufferedOutputStream(mismatchLogOutputStream);) {

			Util.listFilesInPath(remoteDB).forEach(dir1 -> {

				try {
					Util.listFilesInPath(dir1).forEach(dir2 -> {

						try {

							for (Path file : Util.listFilesInPath(dir2)) {

								processFile(file, otherDatabase, bufferedOS);
							}

						} catch (Exception e) {
							throw new RuntimeException(e);
						}
					});
				} catch (Exception e) {
					throw new RuntimeException(e);
				}

			});
		}
	}

	private static void processFile(Path file, IReadableDatabase otherDatabase, OutputStream mismatchLogOutput)
			throws IOException {

		ZipInputStream zis = new ZipInputStream(Files.newInputStream(file));

		do {
			ZipEntry ze = zis.getNextEntry();
			if (ze == null) {
				break;
			}

			String fileContents = new String(zis.readAllBytes());

			for (String line : fileContents.split("\\r?\\n")) {

				String[] tokens = line.split("\\s+");

				String hash = tokens[0];
//				long size = Long.parseLong(tokens[1]);
				String path = line.substring(line.indexOf("\"") + 1, line.lastIndexOf("\""));

//				System.out.println(hash + " " + size + " " + path);

				{
					String matchingShaStringLines = otherDatabase.readDatabaseEntry(hash);

					List<String> result = new ArrayList<>();

					for (String matchLine : matchingShaStringLines.split("\\r?\\n")) {

						if (!matchLine.startsWith(hash + " ")) {
							continue;
						}

						result.add(line);
					}

					if (result.size() == 0) {
						mismatchLogOutput.write(("No match: " + path + "\n").getBytes());
					}

				}

			}

		} while (true);

		zis.close();
	}

}
