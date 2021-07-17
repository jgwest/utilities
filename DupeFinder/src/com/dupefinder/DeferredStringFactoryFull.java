package com.dupefinder;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.RandomAccessFile;

import com.dupefinder.DeferredStringFactoryFull.IDeferredStringFull;

/** Deferred strings should not be serialized to file, or otherwise expected to exist after the JVM has ended. When
 * the factory is closed, or the JVM is shut down, the backing file will be deleted.*/
public class DeferredStringFactoryFull {
		
	long _currPos = 0;
	
	File _sharedFilePath;
	RandomAccessFile _sharedFile;
	
	boolean _isClosed = false;
	
	public DeferredStringFactoryFull(File sharedFile) throws FileNotFoundException {
		_sharedFilePath = sharedFile;
		_sharedFile = new RandomAccessFile(sharedFile, "rw");
		sharedFile.deleteOnExit();
	}
	
	public IDeferredStringFull createString(String str) throws IOException {
		if(str == null) { return null; }
		
		if(str.length() <= 4) {
			return createUnmappedDeferredString(str);
		} else {
			return createDeferredString(str);
		}
	}
	
	/** Will not be mapped to a file, and the text will be fully resident */
	public UnmappedDeferredString createUnmappedDeferredString(String str) {
		return new UnmappedDeferredString(str);
	}
	
	public IDeferredStringFull createDeferredString(String str) throws IOException {
		
		long positionInFile = _currPos;
		
		_sharedFile.seek(_currPos);
		
		byte[] strBytes = str.getBytes(); 
		
		_sharedFile.writeInt(strBytes.length); _currPos +=4;
		_sharedFile.write(strBytes); _currPos += strBytes.length;
	
		IDeferredStringFull result;
		if(positionInFile > (Integer.MAX_VALUE-100) /** Above the max of a signed int, plus some leeway*/) {
			result = new DeferredStringLongFull(positionInFile, this);
		} else {
			result = new DeferredStringIntFull((int)positionInFile, this);
		}
		
		
		return result;
	}
	
	protected String getStringValue(long positionInFile) throws IOException {
		_sharedFile.seek(positionInFile);
		int bytesToRead = _sharedFile.readInt();
		byte[] barr = new byte[bytesToRead];
		
		int c = _sharedFile.read(barr, 0, barr.length);
		if(c != barr.length) {
			throw new IOException("Invalid number of bytes read");
		}
		
		String str = new String(barr);
		return str;
		
	}
	
	public boolean isClosed() {
		return _isClosed;
	}
	
	public void closeFactory() throws IOException {
		_isClosed = true;
		_sharedFile.close();
		_sharedFilePath.delete();
	}
	
	public static interface IDeferredStringFull {

		public String getValue() throws IOException;

	}

	
}

class UnmappedDeferredString implements IDeferredStringFull {
	
	// Unmapped strings only
	byte[] _strValue;


	public UnmappedDeferredString(String value) {
		_strValue = value.getBytes();
	}

	@Override
	public String getValue() {
		return new String(_strValue);
	}

	@Override
	public int hashCode() {
		return getValue().hashCode();
	}
	@Override
	public boolean equals(Object o) {
		try {
			return getValue().equals(((IDeferredStringFull)o).getValue() );
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
	}
	
}

class DeferredStringIntFull implements IDeferredStringFull {	
	
	DeferredStringFactoryFull _parent;
	
	int _positionInFile;
	
	public DeferredStringIntFull(int positionInFile, DeferredStringFactoryFull parent) {
		_positionInFile = positionInFile;
		_parent = parent;
	}

	
	@Override
	public boolean equals(Object o) {
		String strVal;
		try {
			strVal = getValue();
			
			String eqVal = ((IDeferredStringFull)o).getValue();
			
			return strVal.equals(eqVal);
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
		
	}
	
	@Override
	public int hashCode() {
		try {
			return getValue().hashCode();
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
	}
		
	@Override
	public String getValue() throws IOException {
		return 	_parent.getStringValue(_positionInFile);
	}
}

class DeferredStringLongFull implements IDeferredStringFull {	
	
	DeferredStringFactoryFull _parent;
	long _positionInFile;
	
	public DeferredStringLongFull(long positionInFile, DeferredStringFactoryFull parent) {
		_positionInFile = positionInFile;
		_parent = parent;
	}

	
	@Override
	public boolean equals(Object o) {
		String strVal;
		try {
			strVal = getValue();
			
			String eqVal = ((IDeferredStringFull)o).getValue();
			
			return strVal.equals(eqVal);
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
		
	}
	
	@Override
	public int hashCode() {
		try {
			return getValue().hashCode();
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
	}
		
	@Override
	public String getValue() throws IOException {
		return 	_parent.getStringValue(_positionInFile);
	}
}


