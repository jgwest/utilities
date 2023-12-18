package com.dupedeleter;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.FileTime;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
import java.util.zip.CRC32;

public class DupeDeleter {

	public static void main(String[] args) throws IOException, NoSuchAlgorithmException {

//		if (args.length != 1) {
//			System.out.println("Params: (path to delete duplicates on)");
//			return;
//		}

//		Path dirToRecursivelyScan = Paths.get(args[0]);

		System.out.println("Starting.");

		Map<Long /* file size */, List<Path>> fileMap = Step1
				.runStep(List.of(Path.of("J:\\Nostalgia\\jgw-shared-secure-archive")));

//		Map<Long /* file size */, List<Path>> fileMap = Step1
//				.runStep(List.of(Path.of("C:\\delme\\jgw-shared-secure-archive")));

		for (List<Path> pathsMatchedByFilesize2 : fileMap.values()) {

			HashMap<String /* hash */, List<Path>> hashToFileContentsMap = splitPathsByFileContents(
					pathsMatchedByFilesize2);

			for (Map.Entry<String, List<Path>> entry : hashToFileContentsMap.entrySet()) {

				List<Path> pathsByHash = entry.getValue();

				pathsByHash = pathsByHash.stream().filter(p -> !p.getFileName().toString().endsWith(".dupe"))
						.collect(Collectors.toList());

				if (pathsByHash.size() <= 1) {
					continue;
				}

				boolean report = true;

				report = Files.size(pathsByHash.get(0)) > 1 * 1024 * 1024;

				if (report) {

					Collections.sort(pathsByHash);

//					System.out.println();
//					System.out.println("-----------------");

					for (int x = 0; x < pathsByHash.size(); x++) {
						Path p = pathsByHash.get(x);

						if (p.toString().endsWith(".dupe")) {
							throw new RuntimeException("Unexpected dupe entry");
						}

						if (!p.toString().contains("jgw-shared-secure-archive")) {
							continue;
						}

//						System.out.println("- " + p + " " + Files.size(p));

						// Delete everything but the first in the list
						if (x > 0) {
							System.out.println("Delete: " + p);
//							deleteFile(p);
//							Files.delete(p);
						}

//						if (x != pathsByHash.size() - 1) {
//							System.out.println("Delete: " + p);
////										Files.delete(p);
//						}

					}

				}

			}

//				Collections.sort(paths, (a, b) -> {
//
//					int slashesA = countSlashes(a.toString());
//					int slashedB = countSlashes(b.toString());
//
//					int slashResult = slashedB - slashesA;
//
//					if (slashResult != 0) {
//						return slashResult;
//					}
//
//					return a.toString().compareTo(b.toString());
//
//				});

		}

	}

	private static void deleteFile(Path originalFile) throws IOException, NoSuchAlgorithmException {

		String oldSHA = calculateSHA256(originalFile);

		FileTime lastModified = Files.getLastModifiedTime(originalFile);

		String newPathStr = originalFile.toString() + ".dupe";

		Path newPath = Path.of(newPathStr);

		String newFileContent = "";

		newFileContent += "File Size: " + Files.size(originalFile) + "\r\n";
		newFileContent += "File SHA256: " + oldSHA + "\r\n";
		newFileContent += "Last Modified: " + lastModified + "\r\n";
		Files.write(newPath, newFileContent.getBytes());

		Files.setLastModifiedTime(newPath, lastModified);

		Files.delete(originalFile);

	}

	private static HashMap<String, List<Path>> splitPathsByFileContents(List<Path> paths)
			throws IOException, NoSuchAlgorithmException {

		boolean useCRC32 = false;

		HashMap<String /* file contents hash */, List<Path>> res = new HashMap<>();

		for (Path path : paths) {

			String fileHash;
			if (useCRC32) {
				fileHash = calculateCRC(path);
			} else {
				fileHash = calculateSHA256(path);
			}

			List<Path> subPaths = res.computeIfAbsent(fileHash, a -> new ArrayList<Path>());
			subPaths.add(path);

//			if (useCRC32) {
//				if (previousFileHash == null) {
//					previousFileHash = calculateCRC(path);
//				} else {
//
//					String currCrc = calculateCRC(path);
//
//					if (!currCrc.equals(previousFileHash)) {
//						mismatch = true;
//						break inner;
//					}
//				}
//			} else {
//				if (previousFileHash == null) {
//					previousFileHash = calculateSHA256(path);
//				} else {
//
//					String currFileHash = calculateSHA256(path);
//
//					if (!currFileHash.equals(previousFileHash)) {
//						mismatch = true;
//						break inner;
//					}
//				}
//
//			}

		}

		return res;
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

	public static String calculateSHA256(Path p) throws IOException, NoSuchAlgorithmException {

		MessageDigest digest = MessageDigest.getInstance("SHA-256");
//		digest.di

		byte[] barr = Files.readAllBytes(p);

		byte[] res = digest.digest(barr);

		return bytesToHex(res);

	}

	private static String bytesToHex(byte[] hash) {

		StringBuilder hexString = new StringBuilder();

		for (int x = 0; x < hash.length; x++) {
			String hex = Integer.toHexString(0xff & hash[x]);
			if (hex.length() == 1) {
				hexString.append('0');
			}
			hexString.append(hex);
		}

		return hexString.toString();
	}

	private static String calculateCRC(Path p) throws IOException {
		CRC32 crc = new CRC32();

		InputStream is = Files.newInputStream(p);

		int c = 0;
		while (-1 != (c = is.read(sharedBuffer))) {
			crc.update(sharedBuffer, 0, c);
		}

		return "" + crc.getValue();

	}

	private static final byte[] sharedBuffer = new byte[1024 * 1024 * 128];

}
