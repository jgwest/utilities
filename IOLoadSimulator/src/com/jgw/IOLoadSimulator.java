package com.jgw;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;
import java.util.Queue;
import java.util.Random;
import java.util.concurrent.TimeUnit;

public class IOLoadSimulator {

	public static void main(String[] args) {

		List<File> allFiles = new ArrayList<File>();

		{
			File dir = new File(args[0]);
			Queue<File> q = new LinkedList<File>();


			int numFilesIterated = 0;

			q.offer(dir);
			while (q.size() > 0) {

				File f = q.poll();

				if (f.isDirectory()) {

					File[] fileList = f.listFiles();
					
					if(fileList != null) {
						for (File e : fileList) {
	
							q.offer(e);
	
						}
					}

				} else {

					allFiles.add(f);
					numFilesIterated++;
				}
				
				if(numFilesIterated % 5000 == 0 && numFilesIterated > 0) {
					System.out.println(numFilesIterated);
				}

//				if (numFilesIterated > 50000) {
//					break;
//				}

			}
		}
		
		System.out.println("Iteration done.");

		long bytesRead = 0;
		long filesRead = 0;

		long startTimeInNanos = System.nanoTime();

		byte[] barr = new byte[1024 * 1024];

		Random r = new Random();
		while (true) {
			File f = allFiles.get(r.nextInt(allFiles.size()));

			try {

				FileInputStream fis = new FileInputStream(f);

				int c;

				while (-1 != (c = fis.read(barr))) {
					bytesRead += c;
				}
				filesRead++;

				fis.close();

			} catch (FileNotFoundException e) {
//				e.printStackTrace();
			} catch (IOException e) {
//				e.printStackTrace();
			}

			if (filesRead % 3000 == 0) {
				long mbRead2 = bytesRead / (1024l*1024l);
				
				double mbRead = ((double) bytesRead / (1024d * 1024d));
				
				long mbRead2PerSecond = mbRead2
						/ (TimeUnit.SECONDS.convert(System.nanoTime()
								- startTimeInNanos, TimeUnit.NANOSECONDS)); 
				
				double mbReadPerSecond = mbRead
						/ (TimeUnit.SECONDS.convert(System.nanoTime()
								- startTimeInNanos, TimeUnit.NANOSECONDS));

				System.out.println(filesRead + " " + mbReadPerSecond+ " "+mbRead2PerSecond);
			}

		}

	}

}
