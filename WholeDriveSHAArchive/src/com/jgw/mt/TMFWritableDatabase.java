package com.jgw.mt;

import java.io.FileOutputStream;
import java.io.IOException;
import java.io.UncheckedIOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicLong;
import java.util.stream.Collectors;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import com.jgw.util.Util;

public class TMFWritableDatabase implements IWritableDatabase {

	private final Path rootDir;

	private final AtomicLong fileNumber = new AtomicLong(0);

	public TMFWritableDatabase(Path rootDir) {
		this.rootDir = rootDir;
	}

	private String extractHash(String str) {

		// 1) remove .txt
		String contents = str.substring(0, str.lastIndexOf(".txt"));

		// 2) after '-'
		contents = contents.substring(contents.indexOf("-") + 1);

		return contents;

	}

	public void complete2() throws IOException {

		Util.listFilesInPath(rootDir).forEach(dir1 -> {

			try {
				Util.listFilesInPath(dir1).forEach(dir2 -> {

					try {

						List<Path> paths = Util.listFilesInPath(dir2).stream()
								.filter(e -> e.getFileName().toString().endsWith(".txt")).sorted((a, b) -> {
									return extractHash(a.getFileName().toString())
											.compareTo(extractHash(b.getFileName().toString()));
								}).collect(Collectors.toList());

						List<Path> currentGroup = new ArrayList<>();
						String lastHash = null;

						for (Path currPath : paths) {
							String currHash = extractHash(currPath.getFileName().toString());

							if (lastHash == null) {
								lastHash = currHash;

							} else if (!lastHash.equals(currHash)) {

								// combine the full group
								completeGroup(lastHash, currentGroup);

								currentGroup = new ArrayList<>();
							}

							currentGroup.add(currPath);
							lastHash = currHash;
						}

						if (currentGroup.size() != 0) {
							// combine the full group
							completeGroup(lastHash, currentGroup);
						}

					} catch (IOException e) {
						throw new UncheckedIOException(e);
					}

				});
			} catch (IOException e) {
				throw new UncheckedIOException(e);
			}
		});
	}

	private void completeGroup(String shaString, List<Path> currentGroup) {

		List<CombinationEntry> entries = new ArrayList<>();

		currentGroup.forEach(file -> {

			// Sanity check that the file is a text file
			if (!file.getFileName().toString().endsWith(".txt")) {
				return;
			}

			try {
				Files.readAllLines(file).forEach(line -> {

					String[] tokens = line.split("\\s+");

					String sha = tokens[0];
					Long fileSize = Long.parseLong(tokens[1]);
					String filePath = line.substring(line.indexOf("\"") + 1, line.lastIndexOf("\""));

					CombinationEntry ce = new CombinationEntry();
					ce.fileSize = fileSize;
					ce.pathToFile = filePath;
					ce.sha = sha;

					entries.add(ce);

				});

				// Sanity check that the file is a text file
				if (file.getFileName().toString().endsWith(".txt")) {
					Files.delete(file);
				}

			} catch (IOException e) {
				throw new UncheckedIOException(e);
			}

		});

		try {
			combine2(entries, rootDir);
		} catch (IOException | InterruptedException e) {
			throw new RuntimeException(e);
		}

	}

	private static class CombinationEntry {
		long fileSize;
		String pathToFile;
		String sha;
	}

	private static void combine2(List<CombinationEntry> entries, Path rootDir)
			throws IOException, InterruptedException {

		// Here we assume that all the entries will have the same first 6 hash
		// characters, and so we are safe to pass the first entry only.
		Path shaZIPPath = Util.generateOutputPath(entries.get(0).sha, rootDir);
		Files.createDirectories(shaZIPPath.getParent());

		String output = "";
		if (Files.exists(shaZIPPath)) {
			output = Util.readSingleEntryFromZIPFileAsString(shaZIPPath);
		}

		for (CombinationEntry ce : entries) {

			// Sanity test that the combination entry should be written to the same SHA zip
			Path shaZIPPathForCE = Util.generateOutputPath(ce.sha, rootDir);
			if (!shaZIPPathForCE.equals(shaZIPPath)) {
				throw new RuntimeException("mismatch: " + entries.get(0).sha + " vs " + ce.sha);
			}

			output += ce.sha + " " + ce.fileSize + " \"" + ce.pathToFile.toString() + "\"\n";
		}

		writeToFile(output, shaZIPPath);

	}

	public static void writeToFile(String data, Path file) throws IOException {
		ZipOutputStream zos = new ZipOutputStream(new FileOutputStream(file.toFile()));
		zos.putNextEntry(new ZipEntry("data.txt"));
		zos.write(data.getBytes(StandardCharsets.UTF_8));
		zos.close();
	}

	public void addLineToDatabaseEntry(String shaString, long fileSize, Path pathToFile)
			throws IOException, InterruptedException {

		long fileNum = fileNumber.incrementAndGet();

		Path shaTextPath = generateOutputPath(fileNum, shaString, rootDir);
		Files.createDirectories(shaTextPath.getParent());

		if (Files.exists(shaTextPath)) {
			throw new RuntimeException("this shouldn't happpen.");
		}

		String fileContents = shaString + " " + fileSize + " \"" + pathToFile.toString() + "\"\n";

		Files.writeString(shaTextPath, fileContents);

	}

	private static Path generateOutputPath(long fileNumber, String shaString, Path root) {

		String part1 = shaString.substring(0, 2);
		String part2 = shaString.substring(2, 4);
		String part3 = shaString.substring(4, 6);

		Path newPath = root.resolve(part1).resolve(part2).resolve("" + fileNumber + "-" + part3 + ".txt");

		return newPath;

	}

}
