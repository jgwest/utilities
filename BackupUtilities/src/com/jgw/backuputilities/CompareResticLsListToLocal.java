package com.jgw.backuputilities;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

public class CompareResticLsListToLocal {

	private static final boolean isWindowsSeparator = File.separator.equals("\\");

	public static void main(String[] args) throws IOException {

		if (args.length != 1) {

			System.out.println("Parameter required: (path to restic ls list)");
			return;
		}

		// restic ls --recursive -l (snapshot id) "/(target dir)" > e:\ls-output.txt
		Path lsPath = Paths.get(args[0]);

		long totalLines = Files.lines(lsPath).count();

		long[] count = new long[1];

		Files.lines(lsPath).forEach(line -> {

			count[0]++;

			if (count[0] % 50000 == 0) {
				System.out.println(100d * ((double) count[0] / (double) totalLines));
			}

			if (line.startsWith("snapshot ")) {
				return;
			}

			String[] splitBySpacesArr = line.split("\\s+");

			try {
				long fileSize = Long.parseLong(splitBySpacesArr[3]);

				String pathStr = line.substring(line.indexOf(" /") + 1);

				Path path = convertPathIfNeeded(pathStr);

				if (!Files.isDirectory(path)) {
					if (fileSize != Files.size(path)) {
						System.err.println("Mismatch: " + path);
						System.err.println("- " + line);
					}
				}

			} catch (Throwable t) {
//				t.printStackTrace();
				System.err.println();
				System.err.println(line);
				throw new RuntimeException(t);
			}
		});

	}

	public static Path convertPathIfNeeded(String path) {

		if (!isWindowsSeparator) {
			return Paths.get(path);
		}

		String windowsPath = path.replace("/", "\\");
		windowsPath = windowsPath.substring(1);

		String driveLetter = windowsPath.substring(0, 1);

		if (windowsPath.length() > 2) {
			windowsPath = driveLetter + ":" + windowsPath.substring(1);
		} else {
			windowsPath = driveLetter + ":\\";
		}

		return Paths.get(windowsPath);
	}
}
