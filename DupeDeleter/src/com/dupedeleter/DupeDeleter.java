package com.dupedeleter;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.zip.CRC32;

public class DupeDeleter {

	public static void main(String[] args) throws IOException {

		if (args.length != 1) {
			System.out.println("Params: (path to delete duplicates on)");
			return;
		}

		Path dirToRecursivelyScan = Paths.get(args[0]);

		System.out.println("Starting.");

		Map<Long, List<Path>> fileMap = Step1.runStep(dirToRecursivelyScan);

		for (List<Path> paths : fileMap.values()) {

			boolean mismatch = false;
			try {
				Long previousCrc = null;

				inner: for (Path path : paths) {

					if (previousCrc == null) {
						previousCrc = calculateCRC(path);
					} else {

						Long currCrc = calculateCRC(path);

						if (currCrc.longValue() != previousCrc.longValue()) {
							mismatch = true;
							break inner;
						}

					}
				}

			} catch (Exception e) {
				System.err.println("* Skipping " + paths);
			}

			if (!mismatch) {

				Collections.sort(paths, (a, b) -> {

					int slashesA = countSlashes(a.toString());
					int slashedB = countSlashes(b.toString());

					int slashResult = slashedB - slashesA;

					if (slashResult != 0) {
						return slashResult;
					}

					return a.toString().compareTo(b.toString());

				});

				System.out.println();
				System.out.println("-----------------");

				for (int x = 0; x < paths.size(); x++) {
					Path p = paths.get(x);
					System.out.println("- " + p + " " + Files.size(p));
					if (x != paths.size() - 1) {
						System.out.println("Delete: " + p);
//						Files.delete(p);
					}
				}

			}

		}

	}

	private static int countSlashes(String str) {

		int count = 0;
		for (int x = 0; x < str.length(); x++) {

			char ch = str.charAt(x);

			if (ch == '/' || ch == '\\') {
				count++;
			}
		}

		return count;
	}

	public static void oldMain(String[] args) throws IOException {

		System.out.println("Starting.");

		Map<Key, List<Path>> fileMap = new HashMap<>();

		List<Path> queue = new ArrayList<>();

		queue.add(Paths.get("J:\\Pictures"));

		while (queue.size() > 0) {

			Path outerPath = queue.remove(0);

			try {
				Files.list(outerPath).forEach(f -> {

					if (Files.isDirectory(f)) {
						queue.add(f);
					} else {
						try {
							processFile(f, fileMap);
						} catch (IOException e) {
							throw new RuntimeException(e);
						}
					}

				});
			} catch (Exception e) {
				System.err.println("* Skipping " + outerPath);
			}

		}

		fileMap.entrySet().stream().forEach(e -> {
			if (e.getValue().size() > 1) {
				System.out.println();
				System.out.println("-----------------");
				e.getValue().forEach(p -> {
					System.out.println(p);
				});
			}
		});

	}

	private static long calculateCRC(Path p) throws IOException {
		CRC32 crc = new CRC32();

		InputStream is = Files.newInputStream(p);

		int c = 0;
		while (-1 != (c = is.read(sharedBuffer))) {
			crc.update(sharedBuffer, 0, c);
		}

		return crc.getValue();

	}

	private static final byte[] sharedBuffer = new byte[1024 * 1024 * 128];

	private static void processFile(Path p, Map<Key, List<Path>> fileMap) throws IOException {

		String lfname = p.getFileName().toString().toLowerCase();

		boolean isVideo = lfname.endsWith(".mp4") || lfname.endsWith(".avi") || lfname.endsWith(".wmv")
				|| lfname.endsWith(".mpg") || lfname.endsWith(".mpeg") || lfname.endsWith(".asf")
				|| lfname.endsWith(".mkv") || lfname.endsWith(".webm");

		if (!isVideo) {
			return;
		}

//		boolean isImage = lfname.endsWith(".png") || lfname.endsWith(".jpg") || lfname.endsWith(".jpeg")
//				|| lfname.endsWith(".gif") || lfname.endsWith(".webp");
//
//		if (!isImage) {
//			return;
//		}

		CRC32 crc = new CRC32();

		InputStream is = Files.newInputStream(p);

		int c = 0;
		while (-1 != (c = is.read(sharedBuffer))) {
			crc.update(sharedBuffer, 0, c);
		}

//		byte[] fileContents = Files.readAllBytes(p);

		Key key = new Key(Files.size(p), crc.getValue());

		List<Path> paths = fileMap.computeIfAbsent(key, (k) -> new ArrayList<>());
		paths.add(p);

	}

	private static class Key {
		final long fileSize;
		final long crc;

		public Key(long fileSize, long crc) {
			this.fileSize = fileSize;
			this.crc = crc;
		}

		@Override
		public int hashCode() {
			final int prime = 31;
			int result = 1;
			result = prime * result + (int) (crc ^ (crc >>> 32));
			result = prime * result + (int) (fileSize ^ (fileSize >>> 32));
			return result;
		}

		@Override
		public boolean equals(Object obj) {
			if (this == obj)
				return true;
			if (obj == null)
				return false;
			if (getClass() != obj.getClass())
				return false;
			Key other = (Key) obj;
			if (crc != other.crc)
				return false;
			if (fileSize != other.fileSize)
				return false;
			return true;
		}

	}

}
