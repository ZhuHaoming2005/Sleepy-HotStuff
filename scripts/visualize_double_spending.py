#!/usr/bin/env python3
"""
Blockchain Double-Spending Attack Visualizer
Visualizes blockchain forks from multiple nodes to detect double-spending attacks
"""

import json
import sys
import os
import glob
from collections import defaultdict
from typing import Dict, List, Tuple, Set


class Block:
    def __init__(self, node_id: int, height: int, hash_val: str, prehash: str, view: int, tx_count: int, transactions=None):
        self.node_ids = [node_id]  # List of all nodes that have this block
        self.height = height
        self.hash = hash_val
        self.prehash = prehash
        self.view = view
        self.tx_count = tx_count
        self.transactions = transactions or []
    
    def add_node(self, node_id: int):
        """Add a node ID if not already present"""
        if node_id not in self.node_ids:
            self.node_ids.append(node_id)
    
    def __repr__(self):
        hash_str = self.hash[:8] + "..." if self.hash else "Genesis"
        return f"Block(h={self.height}, nodes={self.node_ids}, hash={hash_str})"
    
    def info_str(self):
        hash_str = self.hash[:12] + "..." if self.hash else "Genesis"
        nodes_str = ",".join(map(str, sorted(self.node_ids)))
        return f"Height {self.height} | Nodes [{nodes_str}] | Hash: {hash_str} | TXs: {self.tx_count}"
    
    def tx_details(self):
        """Return transaction details as a list of strings"""
        details = []
        for tx in self.transactions:
            if tx.get('from') or tx.get('to'):
                from_addr = tx.get('from', '')[:8] if tx.get('from') else 'Coinbase'
                to_addr = tx.get('to', '')[:8] if tx.get('to') else 'Unknown'
                value = tx.get('value', 0)
                details.append(f"    {from_addr} → {to_addr}: {value}")
        return details


def load_blockchain(filepath: str, node_id: int) -> List[Block]:
    """Load blockchain data from JSON file"""
    try:
        with open(filepath, 'r') as f:
            data = json.load(f)
        
        blocks = []
        for block_data in data['blocks']:
            block = Block(
                node_id=node_id,
                height=block_data['height'],
                hash_val=block_data['hash'],
                prehash=block_data['prehash'],
                view=block_data['view'],
                tx_count=len(block_data['transactions']),
                transactions=block_data['transactions']
            )
            blocks.append(block)
        
        return blocks
    except FileNotFoundError:
        print(f"Warning: File {filepath} not found")
        return []
    except Exception as e:
        print(f"Error loading {filepath}: {e}")
        return []


def build_block_tree(nodes_blocks: Dict[int, List[Block]]) -> Tuple[Dict[str, Block], Dict[str, List[str]], int]:
    """
    Build a tree structure of unique blocks
    Returns: (unique_blocks, children_map, max_height)
    """
    unique_blocks = {}  # hash -> Block
    children_map = defaultdict(list)  # parent_hash -> [child_hash, ...]
    max_height = 0
    
    # Collect unique blocks and build parent-child relationships
    for node_id, blocks in nodes_blocks.items():
        for block in blocks:
            # Use a unique key: combine hash with height for genesis block
            block_key = block.hash if block.hash else f"genesis_{block.height}"
            
            if block_key not in unique_blocks:
                unique_blocks[block_key] = block
                max_height = max(max_height, block.height)
                
                # Build parent-child relationship
                parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
                if parent_key:
                    if block_key not in children_map[parent_key]:
                        children_map[parent_key].append(block_key)
            else:
                # Block already exists, add this node to its node list
                unique_blocks[block_key].add_node(node_id)
                max_height = max(max_height, block.height)
    
    return unique_blocks, children_map, max_height


def visualize_tree(unique_blocks: Dict[str, Block], children_map: Dict[str, List[str]], max_height: int) -> List[str]:
    """
    Create tree visualization of the blockchain
    Returns list of output lines
    """
    # Create mapping from block to its key
    block_to_key = {}
    for key, block in unique_blocks.items():
        block_to_key[id(block)] = key
    
    # Organize blocks by height
    blocks_by_height = defaultdict(list)
    for key, block in unique_blocks.items():
        blocks_by_height[block.height].append((key, block))
    
    # Track column assignment for each block key
    block_columns = {}
    next_column = [0]
    
    output_lines = []
    
    def get_column(block_key, parent_key):
        """Assign column for a block"""
        if block_key in block_columns:
            return block_columns[block_key]
        
        if parent_key and parent_key in block_columns:
            parent_col = block_columns[parent_key]
            siblings = children_map.get(parent_key, [])
            if len(siblings) > 1:
                # Fork: first child stays, others get new columns
                if siblings.index(block_key) == 0:
                    block_columns[block_key] = parent_col
                    return parent_col
                else:
                    col = next_column[0]
                    next_column[0] += 1
                    block_columns[block_key] = col
                    return col
            else:
                # Single child inherits parent column
                block_columns[block_key] = parent_col
                return parent_col
        else:
            # Root or orphan
            col = next_column[0]
            next_column[0] += 1
            block_columns[block_key] = col
            return col
    
    # First pass: assign columns to all blocks
    for height in range(max_height + 1):
        blocks = blocks_by_height.get(height, [])
        blocks = sorted(blocks, key=lambda x: (x[1].prehash or "", x[0]))
        for block_key, block in blocks:
            parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
            get_column(block_key, parent_key)
    
    # Second pass: generate visualization
    for height in range(max_height + 1):
        blocks = blocks_by_height.get(height, [])
        if not blocks:
            continue
        
        blocks = sorted(blocks, key=lambda x: block_columns.get(x[0], 0))
        max_col = max(block_columns[key] for key, _ in blocks)
        
        # Get all active columns (columns that have blocks at this height or will continue)
        active_columns = set()
        for col in range(max_col + 1):
            # Check if this column has a block at this height or continues from previous
            has_block_here = any(block_columns.get(key) == col for key, _ in blocks)
            # Check if this column continues to next height
            continues = False
            for key, block in blocks:
                if block_columns.get(key) == col:
                    # Check if this block has children
                    if children_map.get(key):
                        continues = True
                        break
            if has_block_here or continues:
                active_columns.add(col)
        
        # Draw connection lines between heights
        if height > 0:
            # Get blocks from previous height
            prev_blocks = blocks_by_height.get(height - 1, [])
            prev_keys = {key for key, _ in prev_blocks}
            
            # Check which columns have forks
            fork_parents = set()
            for block_key, block in blocks:
                parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
                if parent_key in prev_keys:
                    siblings = children_map.get(parent_key, [])
                    if len(siblings) > 1:
                        fork_parents.add(parent_key)
            
            # Draw fork lines
            fork_drawn = False
            for block_key, block in blocks:
                parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
                if parent_key in fork_parents:
                    parent_col = block_columns.get(parent_key, 0)
                    col = block_columns[block_key]
                    siblings = children_map.get(parent_key, [])
                    
                    # Only draw fork line for non-first children
                    if siblings.index(block_key) > 0:
                        line = [' '] * (max_col * 2 + 1)
                        line[parent_col * 2] = '|'
                        
                        # Draw diagonal line
                        if col > parent_col:
                            line[col * 2 - 1] = '\\'
                        else:
                            line[col * 2 + 1] = '/'
                        
                        output_lines.append(''.join(line))
                        fork_drawn = True
            
            # Draw continuation lines for all active branches
            line = [' '] * (max_col * 2 + 1)
            for c in range(max_col + 1):
                # Check if this column has a block at current height
                if any(block_columns.get(key) == c for key, _ in blocks):
                    line[c * 2] = '|'
            if '|' in line:
                output_lines.append(''.join(line))
            
            # Draw continuation lines for all active branches (second line)
            line2 = [' '] * (max_col * 2 + 1)
            for c in range(max_col + 1):
                # Check if this column has a block at current height
                if any(block_columns.get(key) == c for key, _ in blocks):
                    line2[c * 2] = '|'
            if '|' in line2:
                output_lines.append(''.join(line2))
        
        # Draw blocks at this height (compact: same height close together)
        for idx, (block_key, block) in enumerate(blocks):
            col = block_columns[block_key]
            line = [' '] * (max_col * 2 + 1)
            
            # Draw vertical lines for other active columns
            for c in range(max_col + 1):
                # All columns that have blocks at this height should show |
                if any(block_columns.get(key) == c for key, _ in blocks):
                    if c != col:
                        line[c * 2] = '|'
            
            # Draw the block
            line[col * 2] = '*'
            
            tree_str = ''.join(line)
            info_str = block.info_str()
            output_lines.append(f"{tree_str}  {info_str}")
            
            # Add transaction details
            tx_details = block.tx_details()
            if tx_details:
                for tx_line in tx_details:
                    # Draw continuation lines for all active columns
                    cont_line = [' '] * (max_col * 2 + 1)
                    for c in range(max_col + 1):
                        # All columns that have blocks at this height should show |
                        if any(block_columns.get(key) == c for key, _ in blocks):
                            cont_line[c * 2] = '|'
                    output_lines.append(f"{''.join(cont_line)}  {tx_line}")
    
    return output_lines


def detect_forks(children_map: Dict[str, List[str]]) -> Tuple[bool, List[str]]:
    """Detect if there are any forks in the blockchain"""
    fork_detected = False
    fork_points = []
    
    for parent_hash, children in children_map.items():
        if len(children) > 1:
            fork_detected = True
            fork_points.append(parent_hash[:12])
    
    return fork_detected, fork_points


def main():
    # Dynamically find all committedBlocks_*.json files
    output_dir = "./etc/output"
    pattern = os.path.join(output_dir, "committedBlocks_*.json")
    files_found = glob.glob(pattern)
    
    if not files_found:
        print(f"Error: No committedBlocks_*.json files found in {output_dir}")
        return 1
    
    # Extract node IDs from filenames
    files = {}
    for filepath in files_found:
        filename = os.path.basename(filepath)
        # Extract node ID from filename like "committedBlocks_0.json"
        try:
            node_id = int(filename.replace("committedBlocks_", "").replace(".json", ""))
            files[node_id] = filepath
        except ValueError:
            print(f"Warning: Skipping file with invalid format: {filename}")
    
    print(f"Found {len(files)} blockchain files")
    print("Loading blockchain data from nodes...")
    nodes_blocks = {}
    
    for node_id, filepath in sorted(files.items()):
        blocks = load_blockchain(filepath, node_id)
        if blocks:
            nodes_blocks[node_id] = blocks
            max_h = max(b.height for b in blocks)
            print(f"  Node {node_id}: {len(blocks)} blocks loaded (max height: {max_h})")
    
    if not nodes_blocks:
        print("Error: No blockchain data loaded!")
        return 1
    
    print("\nBuilding block tree structure...")
    unique_blocks, children_map, max_height = build_block_tree(nodes_blocks)
    print(f"  Total unique blocks: {len(unique_blocks)}")
    print(f"  Max height: {max_height}")
    
    # Detect forks
    has_fork, fork_points = detect_forks(children_map)
    
    # Generate visualization
    print("\nGenerating tree visualization...")
    tree_lines = visualize_tree(unique_blocks, children_map, max_height)
    
    # Prepare output
    output_lines = []
    output_lines.extend(tree_lines)
    
    # Print to console
    print("\n" + "\n".join(output_lines))
    
    # Write to file
    output_file = "./etc/output/blockchain_analysis.txt"
    try:
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write('\n'.join(output_lines))
        print(f"\n Analysis saved to: {output_file}")
    except Exception as e:
        print(f"\n Error saving to file: {e}")
    
    return 1 if has_fork else 0


if __name__ == "__main__":
    sys.exit(main())

