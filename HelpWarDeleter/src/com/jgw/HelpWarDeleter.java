package com.jgw;

import java.io.File;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;
import java.util.Queue;

public class HelpWarDeleter {

	public static void main(String[] args) {

		if (args.length != 1) {
			System.out.println("Requires one argument: (path to scan)");
			return;
		}

		Queue<File> queue = new LinkedList<File>();

		queue.offer(new File(args[0]));

		List<File> matchingFiles = new ArrayList<File>();

		while (queue.size() > 0) {

			File currDir = queue.poll();

			File[] dirList = currDir.listFiles();
			if (dirList != null) {

				for (File curr : dirList) {

					if (curr.isDirectory()) {

						queue.offer(curr);

					} else {

						if (curr.getPath()
								.endsWith("\\.metadata\\.plugins\\com.ibm.ccl.help.preferenceharvester\\help\\help.war")

//								&& curr.length() > 60000000 
//								&& curr.length() < 70000000
						) {

							matchingFiles.add(curr);
						}

					}

				}

			}

		}

		int count = 0;
		for (File match : matchingFiles) {
			System.out.println(count + ") " + match.getPath() + " " + match.length());
			count++;
		}

//		if(count == 0 || count > 100) {
//			System.err.println("count too large/small");
//			return;
//		}

		System.err.println("Deleting...");

		try {
			Thread.sleep(15 * 1000);
		} catch (Exception e) {
			e.printStackTrace();
			return;
		}

//		for (File match : matchingFiles) {
//			match.delete();
//		}

	}

}
