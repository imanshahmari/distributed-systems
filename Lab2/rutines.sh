#!/bin/bash
gnome-terminal\
    --tab\
        --title="TAB 1" -- bash -c "cd rutines; bash node1.sh; $SHELL"
gnome-terminal\
    --tab\
        --title="TAB 2" -- bash -c "cd rutines; bash node2.sh; $SHELL"
gnome-terminal\
    --tab\
        --title="TAB 2" -- bash -c "cd rutines; bash node3.sh; $SHELL"\
