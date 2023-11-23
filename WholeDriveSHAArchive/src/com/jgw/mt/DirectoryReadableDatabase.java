package com.jgw.mt;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

import com.jgw.util.Util;

public class DirectoryReadableDatabase implements IReadableDatabase {
	private final Path rootDir;

	public DirectoryReadableDatabase(Path rootDir) {
		this.rootDir = rootDir;
	}

	@Override
	public String readDatabaseEntry(String shaString) throws IOException {
		Path shaZIPPath = Util.generateOutputPath(shaString, rootDir);

		String output = "";
		if (Files.exists(shaZIPPath)) {
			output = Util.readSingleEntryFromZIPFileAsString(shaZIPPath);
		}

		return output;

	}

}
