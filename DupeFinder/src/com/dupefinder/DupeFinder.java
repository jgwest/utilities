package com.dupefinder;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileWriter;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.text.NumberFormat;
import java.util.HashSet;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Queue;
import java.util.TreeMap;
import java.util.concurrent.TimeUnit;
import java.util.zip.CRC32;

import com.dupefinder.DeferredStringFactoryFull.IDeferredStringFull;

/*
 * Algorithm order is:
 * - sort by:
 * - 1) filename.toLower().hashcode
 * - 2) file size
 * - 3) crc 
 */

public class DupeFinder {

	public static void main(String[] args) throws IOException {

		if (args.length != 2) {
			System.out.println("Params: (directory to analyze) (output text file path)");
			return;
		}

		doWork(args[0], Paths.get(args[1]));
	}

	private static void doWork(String startDirParam, Path outputFile) throws IOException {

		DeferredStringFactoryFull df = new DeferredStringFactoryFull(
				Files.createTempFile("dupe-finder-", ".dictionary").toFile());

		System.out.println();
		System.out.println("Stage 1:");

		File startDir = new File(startDirParam);

		long startTimeInNanos = System.nanoTime();

		HashSet<Integer /* lcase filename hashcode */> filenameMap = findFilesWithSameFilenameByHashCode(startDir);

		System.out.println("stage 1 total time:"
				+ TimeUnit.SECONDS.convert(System.nanoTime() - startTimeInNanos, TimeUnit.NANOSECONDS));

		System.out.println();
		System.out.println("Stage 2 - files processed:");

		// hash code of duped file name -> Path of file that has a filename that hashes to the key
		TreeMap<Integer, List<IDeferredStringFull>> dupeFilenameMap = getMatchingFiles(startDir, filenameMap, df);

		System.out.println();
		System.out.println("Stage 3:");

		final long MIN_FILE_SIZE = 1024 * 1024 * 5;

		System.out.println("- Dupe file entries: " + dupeFilenameMap.size());
		List<List<File>> dupeFiles = findDupesByCRC(dupeFilenameMap, MIN_FILE_SIZE);

		FileWriter fw = new FileWriter(outputFile.toFile());
		for (List<File> l : dupeFiles) {

			for (File f : l) {
				fw.write(f.getPath() + "\r\n");
				// fw.write(f.length() + " "+f.getPath()+"\r\n");
			}
			fw.write("\r\n");

		}

		fw.close();

		System.out.println();
		System.out.println("Done.");
	}

	private static List<List<File>> findDupesByCRC(TreeMap<Integer, List<IDeferredStringFull>> dupeFilenameMap,
			final long minFileSize) throws IOException {

		List<List<File>> dupeFiles = new LinkedList<List<File>>();

		int debugEntriesProcessed = 0;
		int debugFilesProcessed = 0;

		for (Map.Entry<Integer, List<IDeferredStringFull>> e : dupeFilenameMap.entrySet()) {

			Map<Long /* file size */, List<File> /* files with that file size */> fileSizes = sortByFileSize(
					e.getValue());

			for (Map.Entry<Long, List<File>> l : fileSizes.entrySet()) {

				debugFilesProcessed += l.getValue().size();

				if (l.getKey() < minFileSize) {
					continue;
				}

				List<File> filesWithThatSize = l.getValue();

				if (filesWithThatSize.size() <= 1) {
					continue;
				}

				Map<Long, List<File>> crcMap = sortByCRC(filesWithThatSize);

				for (Map.Entry<Long, List<File>> crcEntry : crcMap.entrySet()) {

					if (crcEntry.getValue().size() > 1) {

						LinkedList<File> dupeFileList = new LinkedList<File>();

						for (File f : crcEntry.getValue()) {
							dupeFileList.add(f);
						}
						dupeFiles.add(dupeFileList);

					}

				}

			}

			debugEntriesProcessed++;

			if (debugEntriesProcessed % 200 == 0) {
				System.out.println(debugEntriesProcessed + ") files processed: "
						+ NumberFormat.getInstance().format(debugFilesProcessed));
			}
		}

		return dupeFiles;

	}

	private static TreeMap<Integer /* hash code of duped file name */, List<IDeferredStringFull>> getMatchingFiles(
			File startDir, HashSet<Integer> filenameMap, DeferredStringFactoryFull df) throws IOException {

		TreeMap<Integer /* hash code of duped file name */, List<IDeferredStringFull>> dupeFilenameMap = new TreeMap<>();

		long filesProcessed = 0;

		Queue<File> queue = new LinkedList<File>();

		queue.offer(startDir);

		while (queue.size() > 0) {
			File currDir = queue.poll();

			File[] dirContents = currDir.listFiles();

			if (dirContents == null) {
				continue;
			}

			for (File f : dirContents) {

				if (f.isDirectory()) {
					queue.offer(f);
				} else {

					int hash = f.getName().toLowerCase().hashCode();

					if (filenameMap.contains(hash)) {
						List<IDeferredStringFull> l = dupeFilenameMap.get(hash);
						if (l == null) {
							l = new LinkedList<IDeferredStringFull>();
							dupeFilenameMap.put(hash, l);
						}

						l.add(df.createDeferredString(f.getPath()));

					}

					filesProcessed++;

					if (filesProcessed % 10000 == 0) {
						System.out.println(NumberFormat.getInstance().format(filesProcessed));
					}
				}

			}

		}

		return dupeFilenameMap;
	}

	/** Not currently used. */
	private static TreeMap<Integer, Boolean> sortByFilenameHashcodeMultithread(File startDir) {

		Queue<File> queue = new LinkedList<File>();

		SortByFilenameHashcodeThread[] threads = new SortByFilenameHashcodeThread[Runtime.getRuntime()
				.availableProcessors()];
		for (int x = 0; x < threads.length; x++) {
			SortByFilenameHashcodeThread thread = new SortByFilenameHashcodeThread(null, x);
			threads[x] = thread;
			thread.start();
		}

		synchronized (queue) {
			queue.offer(startDir);
		}

		boolean continueWork = true;

		while (continueWork) {

			synchronized (queue) {

				if (queue.size() == 0) {
					boolean areAllThreadsEmpty = true;
					for (SortByFilenameHashcodeThread thread : threads) {
						if (!thread.isCurrentlyEmpty()) {
							areAllThreadsEmpty = false;
							break;
						}
					}

					continueWork = !areAllThreadsEmpty;
				}

			}

			if (continueWork) {
				try {
					TimeUnit.MILLISECONDS.sleep(50);
				} catch (InterruptedException e) {
					throw new RuntimeException(e);
				}
			}

		}

		// Shut down the threads and wait for results

		for (SortByFilenameHashcodeThread thread : threads) {
			thread.setContinueRunning(false);
		}

		boolean allResultsAvailable = false;

		while (!allResultsAvailable) {
			allResultsAvailable = true;

			synchronized (queue) {
				for (SortByFilenameHashcodeThread thread : threads) {
					if (thread.getResultFilenameMap() == null) {
						allResultsAvailable = false;
						break;
					}
				}

			}

			if (!allResultsAvailable) {
				try {
					TimeUnit.MILLISECONDS.sleep(50);
				} catch (InterruptedException e) {
					throw new RuntimeException(e);
				}
			}
		}

		// Merge the results
		TreeMap<Integer, Boolean> result = new TreeMap<Integer, Boolean>();
		synchronized (queue) {
			for (SortByFilenameHashcodeThread thread : threads) {
				TreeMap<Integer, Boolean> partialResult = thread.getResultFilenameMap();

				for (Map.Entry<Integer, Boolean> entry : partialResult.entrySet()) {

					Boolean existingResult = result.get(entry.getKey());
					if (existingResult == null || existingResult == false) {
						result.put(entry.getKey(), entry.getValue());
					}

				}

			}
		}

		return result;

	}

	private static HashSet<Integer> findFilesWithSameFilenameByHashCode(File startDir) {

		long matchesFound = 0;
		long filesProcessed = 0;

		long debugCurrSize = 0;

		TreeMap<Integer /* hash code */, Boolean> filenameMap = new TreeMap<>();
		Queue<File> queue = new LinkedList<File>();

		queue.offer(startDir);

		while (queue.size() > 0) {

			File currDir = queue.poll();

			File[] dirContents = currDir.listFiles();
			if (dirContents == null) {
				continue;
			}

			for (File f : dirContents) {

				if (f.isDirectory()) {
					queue.offer(f);
				} else {

					int lnameHash = f.getName().toLowerCase().hashCode();

					Boolean e = filenameMap.get(lnameHash);
					if (e == null) {
						debugCurrSize += 8;
						filenameMap.put(lnameHash, false);
					} else {
						// A filename hash is a dupe if it's seen twice
						filenameMap.put(lnameHash, true);
						matchesFound++;
					}

					filesProcessed++;

					if (filesProcessed % 10000 == 0) {
						NumberFormat nf = NumberFormat.getInstance();
						System.out.println("curr-size: " + nf.format(debugCurrSize) + "   filesProcessed:"
								+ nf.format(filesProcessed) + "   matchesFound:" + nf.format(matchesFound));
					}

				}

			}

		}
		queue = null;

		HashSet<Integer> result = new HashSet<>();
		filenameMap.entrySet().stream().filter(e -> e.getValue()).forEach(e -> {
			result.add(e.getKey());
		});

		return result;

	}

	private static Map<Long, List<File>> sortByCRC(List<File> filePaths) throws IOException {

		Map<Long, List<File>> crcMap = new TreeMap<>();
		for (File f : filePaths) {

			long crc = calculateCRC(f);

			List<File> files = crcMap.get(crc);
			if (files == null) {
				files = new LinkedList<File>();
				crcMap.put(crc, files);
			}
			files.add(f);
		}

		return crcMap;
	}

	private static long calculateCRC(File f) throws IOException {

		byte[] barr = new byte[1024 * 256];

		FileInputStream fis = new FileInputStream(f);

		CRC32 crc = new CRC32();
		int c;
		while (-1 != (c = fis.read(barr))) {

			crc.update(barr, 0, c);
		}

		fis.close();

		return crc.getValue();
	}

	private static Map<Long, List<File>> sortByFileSize(List<IDeferredStringFull> filePaths) throws IOException {

		Map<Long, List<File>> fileSizes = new TreeMap<>();
		for (IDeferredStringFull path : filePaths) {

			File f = new File(path.getValue());

			List<File> files = fileSizes.get(f.length());
			if (files == null) {
				files = new LinkedList<File>();
				fileSizes.put(f.length(), files);
			}
			files.add(f);
		}

		return fileSizes;
	}

}

/** Not currently used */
class SortByFilenameHashcodeThread extends Thread {

	private final SortByFilenameHashcodeThread[] GLOBALthreadList;

	private boolean currentlyEmpty = true;

	private boolean continueRunning = true;

	private TreeMap<Integer, Boolean> resultFilenameMap = null;

	private final int debugThisThreadNum;

	private final Queue<File> sharedInputQueue = new LinkedList<File>();

	public SortByFilenameHashcodeThread(SortByFilenameHashcodeThread[] otherThreads, int debugThisThreadNum) {
		this.GLOBALthreadList = otherThreads;
		this.debugThisThreadNum = debugThisThreadNum;
	}

	public void offerWork(Queue<File> inputWork) {
		synchronized (sharedInputQueue) {

			while (inputWork.size() > 0) {
				sharedInputQueue.offer(inputWork.poll());
			}

		}
	}

	@Override
	public void run() {

		DebugCurrStats stats = new DebugCurrStats();
		stats.debugCurrThreadNum = debugThisThreadNum;

		int nextThread = (debugThisThreadNum + 1) % GLOBALthreadList.length; // (int)(Math.random() *
																				// GLOBALthreadList.length) ;

		TreeMap<Integer, Boolean> filenameMap = new TreeMap<>();

		Queue<File> localOutputQueue = new LinkedList<File>();

		Queue<File> localInputQueue = new LinkedList<File>();

		while (continueRunning) {

			synchronized (sharedInputQueue) {
				while (sharedInputQueue.size() > 0) {
					localInputQueue.offer(sharedInputQueue.poll());
				}

				currentlyEmpty = (localInputQueue.size() == 0);

			}

			SortByFilenameHashcodeThread destThread;
			destThread = GLOBALthreadList[nextThread];
			nextThread = nextThread + 1 % GLOBALthreadList.length;

			if (localInputQueue.size() > 0) {
				processLocalQueue(localInputQueue, localOutputQueue, filenameMap, stats);

				if (localOutputQueue.size() > 0) {
					destThread.offerWork(localOutputQueue);
				}

			} else {
				try {
					TimeUnit.MILLISECONDS.sleep(50);
				} catch (InterruptedException e) {
					throw new RuntimeException(e);
				}
			}
		}

		synchronized (this) {
			resultFilenameMap = filenameMap;
		}

	}

	public TreeMap<Integer, Boolean> getResultFilenameMap() {
		synchronized (this) {
			return resultFilenameMap;
		}
	}

	public void setContinueRunning(boolean continueRunning) {
		synchronized (this) {
			this.continueRunning = continueRunning;
		}
	}

	public boolean isCurrentlyEmpty() {
		synchronized (this) {
			return currentlyEmpty;
		}
	}

	private static void processLocalQueue(Queue<File> localQueue, Queue<File> outputQueue,
			TreeMap<Integer, Boolean> filenameMap, DebugCurrStats stats) {
		while (localQueue.size() > 0) {

			File currDir = localQueue.poll();

			File[] dirContents = currDir.listFiles();
			if (dirContents != null) {

				for (File f : dirContents) {

					if (f.isDirectory()) {
						outputQueue.offer(f);
					} else {

						int lnameHash = f.getName().toLowerCase().hashCode();

						Boolean e = filenameMap.get(lnameHash);
						if (e == null) {
							stats.currSize += 8;
							filenameMap.put(lnameHash, false);
						} else {
							filenameMap.put(lnameHash, true);
							stats.matchesFound++;
						}

						stats.filesProcessed++;

						if (stats.filesProcessed % 10000 == 0) {
							NumberFormat nf = NumberFormat.getInstance();
							System.out.println(stats.debugCurrThreadNum + ") curr-size: " + nf.format(stats.currSize)
									+ "   filesProcessed:" + nf.format(stats.filesProcessed) + "   matchesFounded:"
									+ nf.format(stats.matchesFound)); // / 1024 / 1024);
						}

					}

				}

			}

		}

	}

	private static class DebugCurrStats {
		private long matchesFound = 0;
		private long filesProcessed = 0;

		private long currSize = 0;

		private int debugCurrThreadNum;

	}
}
