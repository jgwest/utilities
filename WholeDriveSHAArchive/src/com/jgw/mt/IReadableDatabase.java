package com.jgw.mt;

import java.io.IOException;

public interface IReadableDatabase {

	public String readDatabaseEntry(String shaString) throws IOException;

}
