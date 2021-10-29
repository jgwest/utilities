package com.jgw;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

import com.jgw.Node.NodeType;
import com.jgw.internal.Data;
import com.jgw.internal.DataBackup;

public class GraphNodeRemover {

	public static void analyze(Data d) {

		Node topLevelDataNode = new Node();
		topLevelDataNode.type = NodeType.DATA;
		topLevelDataNode.setObj(d);

		Data topLevelData = (Data) topLevelDataNode.getObj();
		List<DataBackup> dbList = topLevelData.getDataBackups();

		boolean passwordBackupNeeded = false;

		for (DataBackup db : dbList) {

			if (!db.isUnencrypted()) {
				passwordBackupNeeded = true;
			}

			List<Data> passwordBackups = db.getPasswordBackups();
			for (Data pb : passwordBackups) {

				Node tempNode = new Node();
				tempNode.type = NodeType.PASSWORD_BACKUP;
				tempNode.setObj(pb);

				if (!topLevelDataNode.children.contains(tempNode) && pb != topLevelData) {
					topLevelDataNode.children.add(tempNode);
				}

			}
		}

		if (!passwordBackupNeeded) {
			System.out.println("(No password needed: " + d.getName() + ")");
			return;
		}

		// Remove tag dupes
		for (Iterator<Node> it = topLevelDataNode.children.iterator(); it.hasNext();) {

			Node childNode = it.next();

			Data childLevelData = (Data) childNode.getObj();

			if (childLevelData.getTag().equalsIgnoreCase(topLevelData.getTag())) {
				it.remove();
			}
		}

		// Check for errors
		boolean errorReported = false;

		if (topLevelDataNode.children.size() <= 1) {

			System.err.println("* Data node " + topLevelData.getName() + " has insufficent password backups:");
			System.err.println("\t- " + topLevelDataNode);

			for (Node child : topLevelDataNode.children) {
				System.err.println("\t\to " + child.toString());
			}

			errorReported = true;

		}

		if (errorReported) {
			System.out.println();
		}

//		System.out.println("----------------------------");
//		
//		System.out.println();

	}

//	private static void analyze(DataContext dc) {
//		List<Data> list = dc.getAllData();
//	
//		List<Node> dataToNodeList = new ArrayList<Node>();
//		
//		for(Data d : list) {
//			Node n = new Node();
//			n.type = NodeType.DATA;
//			n.setObj(d);
//			dataToNodeList.add(n);
//		}
//		
//		for(Node topLevelDataNode : dataToNodeList) {
//			
//			Data topLevelData = (Data)topLevelDataNode.getObj();
//			List<DataBackup> dbList = topLevelData.getDataBackups();
//			
//			for(DataBackup db : dbList) {
//				
//				List<Data> passwordBackups = db.getPasswordBackups();
//				for(Data pb : passwordBackups) {
//					
//					Node tempNode = new Node();
//					tempNode.type = NodeType.PASSWORD_BACKUP;
//					tempNode.setObj(pb);
//					
//					
//					if(!topLevelDataNode.children.contains(tempNode) && pb != topLevelData) {
//						topLevelDataNode.children.add(tempNode);
//					}
//					
//				}
//			}
//			
//		}
//		
//		
//		// Remove tag dupes
//		for(Node topLevelDataNode : dataToNodeList) {
//			Data topLevelData = (Data)topLevelDataNode.getObj();
//			
//			for(Iterator<Node> it = topLevelDataNode.children.iterator(); it.hasNext(); ) {
//				
//				Node childNode = it.next();
//				
//				Data childLevelData = (Data)childNode.getObj();
//				
//				if(childLevelData.getTag().equalsIgnoreCase(topLevelData.getTag())) {
//					it.remove();
//				}
//			}
//			
//		}
//		
//		
//		System.out.println("----------------------------");
//		
//		System.out.println("Password Backup Relationships:");
//		
//		for(Node topLevelDataNode : dataToNodeList) {
//			System.out.println();
//			System.out.println(topLevelDataNode);
//			
//			for(Node child : topLevelDataNode.children) {
//				System.out.println("  "+child.toString());
//			}
//			
//		}
//		
//		System.out.println();
//		
//		
//	}

}

class Node {
	enum NodeType {
		DATA, PASSWORD_BACKUP
	};

	NodeType type;
	private Object obj;
	List<Node> children = new ArrayList<Node>();

	@Override
	public boolean equals(Object eqObject) {
		if (!(eqObject instanceof Node)) {
			return false;
		}

		Node other = (Node) eqObject;

		return other.type == type && other.obj.equals(obj);
	}

	@Override
	public String toString() {
		return obj.toString();
	}

	public Object getObj() {
		return obj;
	}

	public void setObj(Object obj) {
		if (obj instanceof Node) {
			throw new RuntimeException("Nope.");
		}
		this.obj = obj;
	}
}