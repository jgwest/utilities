
package com.jgw;

import java.io.File;

public class BugArchiver {

	static String[] _extensions = new String[] { "pch", "sbr", "index", "pdb", "obj", "trcaxmi", "ncb", "bsc", "tis",
			"fdt" /* , "pack" */ };

	/**
	 * @param args
	 */
	public static void main(String[] args) {
		
		if(args.length != 1) {
			System.out.println("Requires one argument: (path to archive)");
			return;
		}

		recurseDirectory(new File(args[0]));
		
	}

	public static void recurseDirectory(File dir) {

		for (File f : dir.listFiles()) {

			if (f.isDirectory()) {
				recurseDirectory(f);
			} else if (f.isFile()) {
				recurseFile(f);
			}
		}

	}

	public static void recurseFile(File file) {
		String name = file.getName();

		int dotPos = name.lastIndexOf(".");

		if (dotPos == -1) {
			return;
		}

		String extension = name.substring(dotPos + 1);

//		if (file.getName().equals("help.war")) {
//			delete(file);
//		}

		for (String ext : _extensions) {
			if (ext.equalsIgnoreCase(extension)) {
				delete(file);
				break;
			}
		}

	}

	private static void delete(File file) {
		System.out.println("Deleting " + file.getPath());
//		file.delete();
	}
}
