package com.jgw.mt;

import java.io.IOException;
import java.nio.file.Path;

public interface IWritableDatabase {

	public void addLineToDatabaseEntry(String shaString, long fileSize, Path pathToFile)
			throws IOException, InterruptedException;
}
