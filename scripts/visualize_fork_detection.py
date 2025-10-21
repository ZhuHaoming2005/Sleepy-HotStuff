#!/usr/bin/env python3
"""
Single Node Fork Detection Visualizer
Visualizes committed blocks (main chain) and received-but-not-committed blocks (side chain)
"""

import json
import sys
import os
from collections import defaultdict
from typing import Dict, List, Tuple, Set
from datetime import datetime


# ANSI Color codes for terminal output
class Colors:
    RESET = '\033[0m'
    BOLD = '\033[1m'
    
    # Text colors
    RED = '\033[91m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    MAGENTA = '\033[95m'
    CYAN = '\033[96m'
    WHITE = '\033[97m'
    GRAY = '\033[90m'
    
    # Background colors
    BG_RED = '\033[101m'
    BG_GREEN = '\033[102m'
    BG_YELLOW = '\033[103m'


class Block:
    def __init__(self, node_id: int, height: int, hash_val: str, prehash: str, view: int, tx_count: int, transactions=None, is_committed=False):
        self.node_id = node_id
        self.height = height
        self.hash = hash_val
        self.prehash = prehash
        self.view = view
        self.tx_count = tx_count
        self.transactions = transactions or []
        self.is_committed = is_committed  # True if committed, False if only received
    
    def __repr__(self):
        hash_str = self.hash[:8] + "..." if self.hash else "Genesis"
        status = "Committed" if self.is_committed else "Received"
        return f"Block(h={self.height}, hash={hash_str}, {status})"
    
    def info_str(self, colored=True):
        hash_str = self.hash[:12] + "..." if self.hash else "Genesis"
        
        if colored:
            if self.is_committed:
                status = f"{Colors.BOLD}{Colors.GREEN}COMMITTED{Colors.RESET}"
                hash_color = Colors.GREEN
            else:
                status = f"{Colors.BOLD}{Colors.YELLOW}RECEIVED{Colors.RESET}"
                hash_color = Colors.YELLOW
            
            return (f"{Colors.CYAN}Height {self.height}{Colors.RESET} | "
                   f"{Colors.BLUE}Replica {self.node_id}{Colors.RESET} | "
                   f"[{status}] | "
                   f"Hash: {hash_color}{hash_str}{Colors.RESET} | "
                   f"{Colors.MAGENTA}TXs: {self.tx_count}{Colors.RESET}")
        else:
            status = "COMMITTED" if self.is_committed else "RECEIVED"
            return f"Height {self.height} | Replica {self.node_id} | [{status}] | Hash: {hash_str} | TXs: {self.tx_count}"
    
    def tx_details(self, colored=True):
        """Return transaction details as a list of strings"""
        details = []
        for tx in self.transactions:
            if tx.get('from') or tx.get('to'):
                from_addr = tx.get('from', '')[:8] if tx.get('from') else 'Coinbase'
                to_addr = tx.get('to', '')[:8] if tx.get('to') else 'Unknown'
                value = tx.get('value', 0)
                timestamp = tx.get('timestamp', 0)
                
                # Format timestamp
                if timestamp > 0:
                    dt = datetime.fromtimestamp(timestamp / 1000.0)
                    time_str = dt.strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]  # Show milliseconds
                else:
                    time_str = "N/A"
                
                if colored:
                    details.append(
                        f"    {Colors.GRAY}{from_addr}{Colors.RESET} "
                        f"{Colors.WHITE}→{Colors.RESET} "
                        f"{Colors.GRAY}{to_addr}{Colors.RESET}: "
                        f"{Colors.CYAN}{value}{Colors.RESET} | "
                        f"{Colors.MAGENTA}{time_str}{Colors.RESET}"
                    )
                else:
                    details.append(f"    {from_addr} → {to_addr}: {value} | {time_str}")
        return details


def load_blocks(filepath: str, node_id: int, is_committed: bool) -> List[Block]:
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
                transactions=block_data['transactions'],
                is_committed=is_committed
            )
            blocks.append(block)
        
        return blocks
    except FileNotFoundError:
        print(f"Warning: File {filepath} not found")
        return []
    except Exception as e:
        print(f"Error loading {filepath}: {e}")
        return []


def build_fork_tree(committed_blocks: List[Block], received_blocks: List[Block]) -> Tuple[Dict[str, Block], Dict[str, List[str]], int, bool]:
    """
    Build a tree structure showing main chain (committed) and side chains (received only)
    Returns: (unique_blocks, children_map, max_height, has_fork)
    """
    unique_blocks = {}  # hash -> Block
    children_map = defaultdict(list)  # parent_hash -> [child_hash, ...]
    max_height = 0
    committed_hashes = set()
    
    # First, collect all committed blocks (main chain)
    for block in committed_blocks:
        block_key = block.hash if block.hash else f"genesis_{block.height}"
        unique_blocks[block_key] = block
        committed_hashes.add(block_key)
        max_height = max(max_height, block.height)
        
        # Build parent-child relationship
        parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
        if parent_key:
            if block_key not in children_map[parent_key]:
                children_map[parent_key].append(block_key)
    
    # Then, add received blocks that are not committed (side chains)
    for block in received_blocks:
        block_key = block.hash if block.hash else f"genesis_received_{block.height}"
        
        # Skip if this block is already committed
        if block_key in committed_hashes:
            continue
        
        # Add only if not already in unique_blocks
        if block_key not in unique_blocks:
            unique_blocks[block_key] = block
            max_height = max(max_height, block.height)
            
            # Build parent-child relationship
            parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
            if parent_key:
                if block_key not in children_map[parent_key]:
                    children_map[parent_key].append(block_key)
    
    # Detect if there's a fork (any parent has more than one child)
    has_fork = any(len(children) > 1 for children in children_map.values())
    
    return unique_blocks, children_map, max_height, has_fork


def visualize_tree(unique_blocks: Dict[str, Block], children_map: Dict[str, List[str]], max_height: int, colored=True, max_display_height=10) -> List[str]:
    """
    Create tree visualization of the blockchain
    Returns list of output lines
    colored: If True, add ANSI color codes for terminal display
    max_display_height: Maximum height to display (default 20)
    """
    # Limit the max_height to display
    display_max_height = min(max_height, max_display_height)
    
    # Organize blocks by height
    blocks_by_height = defaultdict(list)
    for key, block in unique_blocks.items():
        # Only include blocks up to max_display_height
        if block.height <= max_display_height:
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
                # Fork: first child (committed) stays, others (side chains) get new columns
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
    # Prioritize committed blocks to ensure they get lower column numbers (leftmost)
    for height in range(display_max_height + 1):
        blocks = blocks_by_height.get(height, [])
        # Sort: committed blocks first, then by hash
        blocks = sorted(blocks, key=lambda x: (not x[1].is_committed, x[0]))
        for block_key, block in blocks:
            parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
            get_column(block_key, parent_key)
    
    # Second pass: generate visualization
    for height in range(display_max_height + 1):
        blocks = blocks_by_height.get(height, [])
        if not blocks:
            continue
        
        blocks = sorted(blocks, key=lambda x: block_columns.get(x[0], 0))
        max_col = max(block_columns[key] for key, _ in blocks)
        
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
            for block_key, block in blocks:
                parent_key = block.prehash if block.prehash else (f"genesis_{block.height - 1}" if block.height > 0 else None)
                if parent_key in fork_parents:
                    parent_col = block_columns.get(parent_key, 0)
                    col = block_columns[block_key]
                    siblings = children_map.get(parent_key, [])
                    
                    # Only draw fork line for non-first children
                    if siblings.index(block_key) > 0:
                        line = [' '] * (max_col * 2 + 1)
                        
                        # Determine colors: parent block color for |, child block color for \ or /
                        if colored:
                            # Get parent block to determine its color
                            parent_block = unique_blocks.get(parent_key)
                            if parent_block:
                                parent_color = Colors.GREEN if parent_block.is_committed else Colors.YELLOW
                            else:
                                parent_color = Colors.GREEN  # Default to green for genesis
                            
                            # Child block color for the diagonal line
                            child_color = Colors.GREEN if block.is_committed else Colors.YELLOW
                            
                            # | uses parent color, \ or / uses child color
                            line[parent_col * 2] = f"{parent_color}|{Colors.RESET}"
                            
                            # Draw diagonal line with child color
                            if col > parent_col:
                                line[col * 2 - 1] = f"{child_color}\\{Colors.RESET}"
                            else:
                                line[col * 2 + 1] = f"{child_color}/{Colors.RESET}"
                        else:
                            line[parent_col * 2] = '|'
                            
                            # Draw diagonal line
                            if col > parent_col:
                                line[col * 2 - 1] = '\\'
                            else:
                                line[col * 2 + 1] = '/'
                        
                        output_lines.append(''.join(line))
            
            # Draw continuation lines for all active branches (first line)
            line = [' '] * (max_col * 2 + 1)
            for c in range(max_col + 1):
                block_at_col = next((b for key, b in blocks if block_columns.get(key) == c), None)
                if block_at_col:
                    if colored:
                        pipe_color = Colors.GREEN if block_at_col.is_committed else Colors.YELLOW
                        line[c * 2] = f"{pipe_color}|{Colors.RESET}"
                    else:
                        line[c * 2] = '|'
            if any(c != ' ' for c in line):
                output_lines.append(''.join(line))
            
            # Draw continuation lines for all active branches (second line)
            line2 = [' '] * (max_col * 2 + 1)
            for c in range(max_col + 1):
                block_at_col = next((b for key, b in blocks if block_columns.get(key) == c), None)
                if block_at_col:
                    if colored:
                        pipe_color = Colors.GREEN if block_at_col.is_committed else Colors.YELLOW
                        line2[c * 2] = f"{pipe_color}|{Colors.RESET}"
                    else:
                        line2[c * 2] = '|'
            if any(c != ' ' for c in line2):
                output_lines.append(''.join(line2))
        
        # Draw blocks at this height (compact: same height close together)
        for idx, (block_key, block) in enumerate(blocks):
            col = block_columns[block_key]
            line = [' '] * (max_col * 2 + 1)
            
            # Draw vertical lines for other active columns
            for c in range(max_col + 1):
                other_block = next((b for key, b in blocks if block_columns.get(key) == c), None)
                if other_block and c != col:
                    if colored:
                        pipe_color = Colors.GREEN if other_block.is_committed else Colors.YELLOW
                        line[c * 2] = f"{pipe_color}|{Colors.RESET}"
                    else:
                        line[c * 2] = '|'
            
            # Draw the block
            if colored:
                block_color = Colors.GREEN if block.is_committed else Colors.YELLOW
                line[col * 2] = f"{Colors.BOLD}{block_color}*{Colors.RESET}"
            else:
                line[col * 2] = '*'
            
            tree_str = ''.join(line)
            info_str = block.info_str(colored=colored)
            output_lines.append(f"{tree_str}  {info_str}")
            
            # Add transaction details
            tx_details = block.tx_details(colored=colored)
            if tx_details:
                for tx_line in tx_details:
                    # Draw continuation lines for all active columns
                    cont_line = [' '] * (max_col * 2 + 1)
                    for c in range(max_col + 1):
                        active_block = next((b for key, b in blocks if block_columns.get(key) == c), None)
                        if active_block:
                            if colored:
                                pipe_color = Colors.GREEN if active_block.is_committed else Colors.YELLOW
                                cont_line[c * 2] = f"{pipe_color}|{Colors.RESET}"
                            else:
                                cont_line[c * 2] = '|'
                    output_lines.append(f"{''.join(cont_line)}  {tx_line}")
    
    return output_lines


def main():
    # Configuration
    node_id = 2
    output_dir = "./etc/output"
    committed_file = os.path.join(output_dir, f"committedBlocks_{node_id}.json")
    received_file = os.path.join(output_dir, f"receivedBlocks_{node_id}.json")
    
    #print(f"Loading blockchain data for Node {node_id}...")
    
    # Load committed blocks (main chain)
    committed_blocks = load_blocks(committed_file, node_id, is_committed=True)
    if not committed_blocks:
        print(f"Error: No committed blocks loaded from {committed_file}")
        return 1
    #print(f"  Committed blocks: {len(committed_blocks)} (max height: {max(b.height for b in committed_blocks)})")
    
    # Load received blocks
    received_blocks = load_blocks(received_file, node_id, is_committed=False)
    if not received_blocks:
        print(f"Warning: No received blocks loaded from {received_file}")
        received_blocks = []
    else:
        #print(f"  Received blocks: {len(received_blocks)} (max height: {max(b.height for b in received_blocks)})")
        pass
    
    # Build fork tree
    #print("\nBuilding block tree structure...")
    unique_blocks, children_map, max_height, has_fork = build_fork_tree(committed_blocks, received_blocks)
    
    committed_count = sum(1 for b in unique_blocks.values() if b.is_committed)
    received_only_count = sum(1 for b in unique_blocks.values() if not b.is_committed)
    '''
    print(f"  Total unique blocks: {len(unique_blocks)}")
    print(f"  Committed blocks: {committed_count}")
    print(f"  Received-only blocks (side chain): {received_only_count}")
    print(f"  Max height: {max_height}")
    print(f"  Fork detected: {'YES' if has_fork else 'NO'}")
    '''
    # Generate visualization
    #print("\nGenerating tree visualization...")
    
    # Generate colored version for console (limit to height 20)
    tree_lines_colored = visualize_tree(unique_blocks, children_map, max_height, colored=True, max_display_height=10)
    
    # Generate plain version for file (limit to height 20)
    tree_lines_plain = visualize_tree(unique_blocks, children_map, max_height, colored=False, max_display_height=10)
    
    # Print to console with colors
    print("\n" + "\n".join(tree_lines_colored))
    
    # Write to file without colors
    output_file = os.path.join(output_dir, f"fork_analysis_node_{node_id}.txt")
    try:
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write('\n'.join(tree_lines_plain))
        #print(f"\nAnalysis saved to: {output_file}")
    except Exception as e:
        print(f"\nError saving to file: {e}")
    
    return 0


if __name__ == "__main__":
    sys.exit(main())

