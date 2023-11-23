package com.jgw.client;

import java.io.FileInputStream;
import java.io.IOException;
import java.nio.file.Path;

import com.jgw.mt.DirectoryReadableDatabase;
import com.jgw.mt.IReadableDatabase;
import com.jgw.util.Util;

public class FindSingleFileMatches {

	public static void main(String[] args) throws IOException, InterruptedException {

		Path databaseDirPath = Path.of("c:\\database-directory-path");

		IReadableDatabase db = new DirectoryReadableDatabase(databaseDirPath);

		Path fileToSearch = Path.of("c:\\file");

		findAMatchForASingleFile(fileToSearch, db);

	}

	private static void findAMatchForASingleFile(Path fileToSearch, IReadableDatabase db) throws IOException {

		String shaString;
		{
			FileInputStream fis = new FileInputStream(fileToSearch.toFile());
			try {
				shaString = Util.getSHA256(fis);
			} finally {
				fis.close();
			}
		}

		String res = db.readDatabaseEntry(shaString);

		for (String line : res.split("\\r?\\n")) {

			if (!line.startsWith(shaString + " ")) {
				continue;
			}

			System.out.println(line);

		}

	}

}
