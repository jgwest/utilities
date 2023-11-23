package com.jgw.util;

import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import java.util.stream.Stream;
import java.util.zip.ZipEntry;
import java.util.zip.ZipFile;

public class BenchmarkZipRead {

	public static void main(String[] args) throws IOException, InterruptedException {

		zipTest();
	}

	@SuppressWarnings("unused")
	private static void nonZipTest() throws IOException, InterruptedException {

		List<Path> pathList = new ArrayList<>();

		{
			List<Path> stack = new ArrayList<Path>();

			stack.add(Path.of("C:\\db-everything"));

			while (stack.size() > 0) {

				Path currPath = stack.remove(stack.size() - 1);

				Stream<Path> currPathChildren = Files.list(currPath);

				currPathChildren.forEach(childPath -> {
					if (Files.isDirectory(childPath)) {
						stack.add(childPath);
					} else {
						pathList.add(childPath);
					}
				});

				currPathChildren.close();

			}

		}

		Collections.shuffle(pathList);

		System.out.println("Done.");

		long startTimeInNanos = System.nanoTime();

		byte[] barr = new byte[1024 * 1024];

		long total = 0;

		long count = 0;

		for (Path p : pathList) {

			InputStream is = new FileInputStream(p.toFile());

			int c = 0;
			do {
				c = is.read(barr);
				total += c;
			} while (c != -1);

			is.close();

			count++;
			if (count % 100_000 == 0) {
				System.out.println(count);
			}
		}

		System.out.println("-----");
		System.out.println(TimeUnit.SECONDS.convert(System.nanoTime() - startTimeInNanos, TimeUnit.NANOSECONDS));
		System.out.println(total);

	}

	private static void zipTest() throws IOException, InterruptedException {
		ZipFile zf = new ZipFile("c:\\db-everything.zip");

		List<ZipEntry> zeList = new ArrayList<>();

		Map<String, ZipEntry> zeMap = new HashMap<>();

		zf.entries().asIterator().forEachRemaining(e -> {

			zeList.add(e);
			zeMap.put(e.getName(), e);
		});

		Collections.shuffle(zeList);

		System.out.println("Done.");

		long startTimeInNanos = System.nanoTime();

		byte[] barr = new byte[1024 * 1024];

		long total = 0;

		long count = 0;

		for (ZipEntry ze : zeList) {

			InputStream zis = zf.getInputStream(ze);

			int c = 0;
			do {
				c = zis.read(barr);
				total += c;
			} while (c != -1);

			zis.close();

			count++;
			if (count % 100_000 == 0) {
				System.out.println(count);
			}
		}

		zf.close();

		System.out.println("-----");
		System.out.println(TimeUnit.SECONDS.convert(System.nanoTime() - startTimeInNanos, TimeUnit.NANOSECONDS));
		System.out.println(total);
	}

}
