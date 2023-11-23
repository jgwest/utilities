package com.jgw.util;

import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.stream.Collectors;
import java.util.stream.Stream;
import java.util.zip.ZipInputStream;

public class Util {

	public static final long MAX_SIZE = 3 * 1024 * 1024 * 1024l;
	public static final long MIN_SIZE = 1024;

	public static boolean filteredOutByFileSize(long fileSize) {
		return fileSize > Util.MAX_SIZE || fileSize < Util.MIN_SIZE;
	}

	public static List<Path> listFilesInPath(Path p) throws IOException {

		List<Path> res = new ArrayList<>();

		try (Stream<Path> files = Files.list(p)) {
			res.addAll(files.collect(Collectors.toList()));
		}

		return res;
	}

	public static void waitForQueueSize(ThreadPoolExecutor es) {
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

	public static String getSHA256(InputStream is) {
		MessageDigest digest;
		try {
			// originalString.getBytes(StandardCharsets.UTF_8)
			digest = MessageDigest.getInstance("SHA-256");

			byte[] barr = new byte[512 * 1024];
			while (true) {
				int c = is.read(barr);
				if (c == -1) {
					break;
				}
				digest.update(barr, 0, c);
			}

			byte[] encodedhash = digest.digest();
			String res = bytesToHex(encodedhash);
			return res;
		} catch (NoSuchAlgorithmException e) {
			throw new RuntimeException(e);
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
	}

	private static String bytesToHex(byte[] hash) {
		StringBuilder hexString = new StringBuilder(2 * hash.length);
		for (int i = 0; i < hash.length; i++) {
			String hex = Integer.toHexString(0xff & hash[i]);
			if (hex.length() == 1) {
				hexString.append('0');
			}
			hexString.append(hex);
		}
		return hexString.toString();
	}

	public static String readSingleEntryFromZIPFileAsString(Path zipFile) throws IOException {
		ZipInputStream zis = new ZipInputStream(new FileInputStream(zipFile.toFile()));
		zis.getNextEntry();

		byte[] barr = zis.readAllBytes();

		zis.close();

		return new String(barr, StandardCharsets.UTF_8);

	}

	public static Path generateOutputPath(String shaString, Path root) {

		String part1 = shaString.substring(0, 2);
		String part2 = shaString.substring(2, 4);
		String part3 = shaString.substring(4, 6);

		Path newPath = root.resolve(part1).resolve(part2).resolve(part3 + ".zip");

		return newPath;

	}

}
