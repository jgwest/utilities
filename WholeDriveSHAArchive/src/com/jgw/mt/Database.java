package com.jgw.mt;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

import com.jgw.util.Util;

public class Database implements IReadableDatabase {

	private final Path rootDir;

//	private Map<String, ReentrantLock> mapActiveLocks = new HashMap<>();

//	private HashSet<String> activeLocks = new HashSet<>();

	public Database(Path rootDir) {
		this.rootDir = rootDir;
	}

	public String readDatabaseEntry(String shaString) throws IOException {
		Path shaZIPPath = Util.generateOutputPath(shaString, rootDir);

		String output = "";
		if (Files.exists(shaZIPPath)) {
			output = Util.readSingleEntryFromZIPFileAsString(shaZIPPath);
		}

		return output;

	}
//
//	private void addLineToDatabaseEntry(String shaString, long fileSize, Path pathToFile)
//			throws IOException, InterruptedException {
//
//		Path shaZIPPath = Util.generateOutputPath(shaString, rootDir);
//		Files.createDirectories(shaZIPPath.getParent());
//
//		String key = shaZIPPath.toString();
//
//		while (true) {
//			synchronized (activeLocks) {
//				if (!activeLocks.contains(key)) {
//					activeLocks.add(key);
//					break;
//				}
//			}
//			System.out.println("(" + Thread.currentThread().getId() + ") * Waiting: " + shaString);
//			Thread.sleep(1000);
//		}
//
////		ReentrantLock lock;
////		synchronized (mapActiveLocks) {
////			lock = mapActiveLocks.computeIfAbsent(key, e -> new ReentrantLock());
////			lock.lock();
////			mapActiveLocks.put(key, lock);
////		}
//
//		String output = "";
//		if (Files.exists(shaZIPPath)) {
//			output = Util.readSingleEntryFromZIPFileAsString(shaZIPPath);
//		}
//
//		output += shaString + " " + fileSize + " \"" + pathToFile.toString() + "\"\n";
//
//		TMFWritableDatabase.writeToFile(output, shaZIPPath);
//
//		synchronized (activeLocks) {
//			if (!activeLocks.remove(key)) {
//				throw new RuntimeException("This shouldn't happen.");
//			}
//		}
////		lock.unlock();
//
//	}

}
