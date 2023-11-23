package com.jgw.client;

import java.io.FileInputStream;
import java.io.IOException;
import java.io.UncheckedIOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.locks.ReentrantLock;

import com.jgw.mt.DirectoryReadableDatabase;
import com.jgw.mt.IReadableDatabase;
import com.jgw.util.Util;

public class RecursiveScanForMissingOrDupedFiles {

	private static final AtomicLong filesScanned = new AtomicLong(0);

	private static final ReentrantLock consoleLock = new ReentrantLock();

	private static final boolean showMatches = false;

	public static void main(String[] args) throws InterruptedException {

		long startTimeInNanos = System.nanoTime();

		ThreadPoolExecutor es = (ThreadPoolExecutor) Executors.newFixedThreadPool(4);

		Path rootPath = Path.of("c:\\delme\\db");

		Path startingDir = Path.of("j:\\Pictures");

		IReadableDatabase db = new DirectoryReadableDatabase(rootPath);

//		IReadableDatabase db = new ZipReadableDatabase(Path.of("c:\\db\\db.zip"));

		processDirectory(startingDir, es, db);

		es.shutdown();

		System.out.println("* Done iterating over files.");

		es.awaitTermination(Long.MAX_VALUE, TimeUnit.DAYS);

		System.out.println("* Terminated. Elapsed: "
				+ TimeUnit.SECONDS.convert(System.nanoTime() - startTimeInNanos, TimeUnit.NANOSECONDS));

	}

	private static void processDirectory(Path dirParam, ThreadPoolExecutor es, IReadableDatabase db) {

		try {
			Util.listFilesInPath(dirParam).forEach(f -> {

				if (Files.isDirectory(f)) {
					processDirectory(f, es, db);
				} else {

					if (es != null) {
						es.submit(new RunnableTask(f, db));

						Util.waitForQueueSize(es);
					} else {
						processFile(f, db);
					}
				}

			});
		} catch (IOException e) {
			e.printStackTrace();
		}

	}

	private static void processFile(Path fileParam, IReadableDatabase db) {

		try {
			List<String> res = findMatches(fileParam, db);

			if (res.size() == 0) {
				try {
					consoleLock.lock();
					System.out.println("No match: " + fileParam);
				} finally {
					consoleLock.unlock();
				}

			} else {
				if (showMatches) {
					try {
						consoleLock.lock();
						System.out.println();
						System.out.println(fileParam.toString() + ":");
						res.forEach(e -> {
							System.out.println("- " + e);
						});
					} finally {
						consoleLock.unlock();
					}
				}
			}

			long scanned = filesScanned.incrementAndGet();
			if (scanned % 10000 == 0) {

				try {
					consoleLock.lock();
					System.out.println("Files scanned: " + scanned);
				} finally {
					consoleLock.unlock();
				}

			}

		} catch (IOException e) {
			throw new UncheckedIOException(e);
		}

	}

	private static List<String> findMatches(Path file, IReadableDatabase db) throws IOException {

		String shaString;
		FileInputStream fis = new FileInputStream(file.toFile());
		try {
			shaString = Util.getSHA256(fis);
		} finally {
			fis.close();
		}

		String matchingShaStringLines = db.readDatabaseEntry(shaString);

		List<String> result = new ArrayList<>();

		for (String line : matchingShaStringLines.split("\\r?\\n")) {

			if (!line.startsWith(shaString + " ")) {
				continue;
			}

			result.add(line);
		}

		return result;
	}

	private static class RunnableTask implements Runnable {

		private final Path targetPath;
		private final IReadableDatabase db;

		public RunnableTask(Path targetPath, IReadableDatabase db) {
			this.targetPath = targetPath;
			this.db = db;
		}

		@Override
		public void run() {

			long fileSize;
			try {
				fileSize = Files.size(targetPath);
			} catch (IOException e) {
				e.printStackTrace();
				return;
			}

			if (Util.filteredOutByFileSize(fileSize)) {
				return;
			}

			processFile(targetPath, db);
		}
	}
}
