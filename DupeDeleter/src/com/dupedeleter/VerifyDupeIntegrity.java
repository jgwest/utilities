package com.dupedeleter;

import java.io.IOException;
import java.io.UncheckedIOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

public class VerifyDupeIntegrity {

	public static void main(String[] args) throws IOException {

		Path p = Paths.get("J:\\Nostalgia\\jgw-shared-secure-archive");
//		Path p = Paths.get("C:\\delme\\jgw-shared-secure-archive");

		HashSet<String> hashSet = new HashSet<>();

		AtomicInteger count = new AtomicInteger(0);
		{
			List<Path> dirList2 = new ArrayList<Path>();
			dirList2.add(p);

			while (dirList2.size() > 0) {

				Path currentDirectory = dirList2.remove(0);

				Files.list(currentDirectory).forEach(currPathFile -> {

					if (Files.isDirectory(currPathFile)) {
						dirList2.add(currPathFile);
						return;
					}

					if (currPathFile.getFileName().toString().endsWith(".dupe")) {
						return;
					}

					try {
						String sha = DupeDeleter.calculateSHA256(currPathFile);
						hashSet.add(sha);
						if (count.incrementAndGet() % 1000 == 0) {
							System.out.println(count.get());
						}
					} catch (NoSuchAlgorithmException | IOException e) {
						throw new RuntimeException(e);
					}

				});

			}
		}

		System.out.println("Hash set complete: " + hashSet.size());

		ArrayList<Path> dirList = new ArrayList<>();
		dirList.add(p);

		AtomicInteger matchCount = new AtomicInteger(0);
		AtomicInteger dupeCount = new AtomicInteger(0);

		while (dirList.size() > 0) {

			Path currentDirectory = dirList.remove(0);

			Files.list(currentDirectory).forEach(currPathFile -> {

				if (Files.isDirectory(currPathFile)) {
					dirList.add(currPathFile);
					return;
				}

				if (!currPathFile.getFileName().toString().endsWith(".dupe")) {
					return;
				}

				dupeCount.incrementAndGet();

				List<String> dupeFileContents;
				try {
					dupeFileContents = Files.readAllLines(currPathFile);
				} catch (IOException e) {
					throw new UncheckedIOException(e);
				}

				String shaLine = dupeFileContents.stream().filter(line -> line.startsWith("File SHA256:")).findFirst()
						.get();

				String sha = shaLine.substring(shaLine.lastIndexOf(" ") + 1);

				if (!hashSet.contains(sha)) {
					System.out.println("Missing SHA: " + sha);
				} else {
					matchCount.incrementAndGet();
				}

			});
		}

		System.out.println("Total matches: " + matchCount.intValue() + " of " + dupeCount.intValue());
	}

}
