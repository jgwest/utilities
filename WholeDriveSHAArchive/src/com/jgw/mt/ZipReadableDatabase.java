package com.jgw.mt;

import java.io.IOException;
import java.io.InputStream;
import java.io.UncheckedIOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;
import java.util.zip.ZipEntry;
import java.util.zip.ZipFile;
import java.util.zip.ZipInputStream;

public class ZipReadableDatabase implements IReadableDatabase {

	private final ZipFile zipFile;
	private final Map<String, ZipEntry> zipEntryMap = new HashMap<>();

	public ZipReadableDatabase(Path zipFile) {

		try {
			this.zipFile = new ZipFile(zipFile.toFile());
			this.zipFile.entries().asIterator().forEachRemaining(zipEntry -> {
				zipEntryMap.put(zipEntry.getName(), zipEntry);
			});
		} catch (IOException e1) {
			throw new UncheckedIOException(e1);
		}

	}

	@Override
	public String readDatabaseEntry(String shaString) throws IOException {
//		Path shaZIPPath = Database.generateOutputPath(shaString, zipRootFolder);

		String part1 = shaString.substring(0, 2);
		String part2 = shaString.substring(2, 4);
		String part3 = shaString.substring(4, 6);

		String shaZIPPath = part1 + "/" + part2 + "/" + part3 + ".zip";

		String res = "";

		ZipEntry ze = zipEntryMap.get(shaZIPPath);

		if (ze == null) {
			return res;
		}

		try (InputStream fis = zipFile.getInputStream(ze)) {

			try (ZipInputStream zis = new ZipInputStream(fis)) {
				zis.getNextEntry();

				byte[] barr = zis.readAllBytes();

				zis.close();

				res = new String(barr, StandardCharsets.UTF_8);
			}

		}

		return res;
	}

}
