package com.jgw.client;

import java.io.FileInputStream;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;
import java.util.Queue;
import java.util.concurrent.Executors;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;

import com.jgw.mt.IWritableDatabase;
import com.jgw.mt.TMFWritableDatabase;
import com.jgw.util.Util;

public class GenerateMain {

	private static final AtomicLong filesProcessed = new AtomicLong(0);

	public static void main(String[] args) throws IOException, InterruptedException {

		if (args.length != 2) {
			System.out.println("(database root path) (path to scan)");
			return;
		}

		String rootPathStr = args[0];

		String pathToScan = args[1];

//		String rootPathStr = "c:\\db-everything";

		List<Path> pathsToScan = new ArrayList<>();

		pathsToScan.add(Paths.get(pathToScan));

		generateDatabase(pathsToScan, Paths.get(rootPathStr));

	}

	private static void generateDatabase(List<Path> pathsToScan, Path dbRootPath)
			throws IOException, InterruptedException {

		TMFWritableDatabase db = new TMFWritableDatabase(dbRootPath);

		ThreadPoolExecutor es = (ThreadPoolExecutor) Executors.newFixedThreadPool(2);

		for (Path targetPath : pathsToScan) {
			processDirectory(targetPath, es, db);
		}

		if (es != null) {
			es.shutdown();
		}

		System.out.println("* Done iterating over files.");

		if (es != null) {
			es.awaitTermination(Long.MAX_VALUE, TimeUnit.DAYS);
		}

		System.out.println("* Completing.");

		db.complete2();

		System.out.println("* Terminated.");

	}

	private static void processDirectory(Path targetDir, ThreadPoolExecutor es, IWritableDatabase db)
			throws IOException {

		Queue<Path> queue = new LinkedList<>();

		queue.add(targetDir);

		while (queue.size() > 0) {

			try {

				Path p = queue.remove();

				if (Files.isSymbolicLink(p)) {
					continue;
				}

				if (!Files.isDirectory(p)) {
					throw new RuntimeException(p.toString() + " is a file.");
				}

				List<Path> dirPaths = Util.listFilesInPath(p);

				for (Path dirPath : dirPaths) {

					if (Files.isSymbolicLink(dirPath)) {
						continue;
					}

					if (Files.isDirectory(dirPath)) {
						queue.add(dirPath);
					} else {
						if (es != null) {
							es.submit(new RunnableTask(dirPath, db));
							waitForQueueSize(es);
						} else {

							try {
								RunnableTask rt = new RunnableTask(dirPath, db);
								rt.run();
							} catch (Exception e2) {
								System.err.println(
										targetDir.toString() + " " + e2.getClass().getName() + ": " + e2.getMessage());
							}

						}
					}

				}

			} catch (IOException e) {
				System.err.println(targetDir.toString() + " " + e.getClass().getName() + ": " + e.getMessage());
			}

		}

	}

	private static void waitForQueueSize(ThreadPoolExecutor es) {
		boolean queueTooLarge = false;

		int queueSize = es.getQueue().size();
		while (queueSize > 5000) {
			queueTooLarge = true;
			try {
				Thread.sleep(500);
			} catch (InterruptedException e) {
				throw new RuntimeException(e);
			}

			queueSize = es.getQueue().size();

		}
		if (queueTooLarge) {

			if (queueSize < 1000) {
				System.err.println("Max queue size is too small!");
			}
		}

	}

	private static class RunnableTask implements Runnable {

		private final Path targetFile;

		private final IWritableDatabase db;

		public RunnableTask(Path targetFile, IWritableDatabase db) {
			this.targetFile = targetFile;
			this.db = db;
		}

		@Override
		public void run() {

			long fileSize;
			try {
				fileSize = Files.size(targetFile);
				if (Util.filteredOutByFileSize(fileSize)) {
					return;
				}

				FileInputStream fis = new FileInputStream(targetFile.toFile());
				String shaString;
				try {
					shaString = Util.getSHA256(fis);
				} finally {
					fis.close();
				}

				db.addLineToDatabaseEntry(shaString, fileSize, targetFile);

				long val = filesProcessed.incrementAndGet();
				if (val % 1000 == 0) {
					System.out.println(val);
				}

			} catch (Exception e) {
				throw new RuntimeException(e);
			}

		}

	}

}
