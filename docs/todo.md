__________________________________________
#
# # # **IMPORTANT** # # # 

- Encounters seem to be too skewed towards ranged threats. Using the threat layer visualization as baseline. Rectify that. 

- There has to be a tradeoff between casting spells and moving squads/engaging in combat. Doing both will feel too mechanical. 

- type EventType int in Overworld types is growing too large


 - System for gaining resources. Whether it's through battle or territory tbd


- Need some calvary untis

- Overworld node spawning is far too clustered together. All nodes spawn close to eachother. 


- Clear overowlrd encounter reward system. 

- Lots of Empty Claude relate directories. See if there are any issues when removing them

- Consider using an interface for intent in overworld

- Need to be able to equip items to squads

-  ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  

- Re-neable victory conditions in tickmanager.go. They are disabled for testing

- encounter service should not have mode coordinator. It overlaps too much with what the GUI is doing


- calculateEntryIndexAtPosition in squadeditor_movesquad.go is not accurately calculating the position to move the squad to. Check if there is an ebitenui var (such as entryselected) you can use instead of calculating with pixel positions


- Handle Entity Cleanup

- JSON file error handling


- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer

- Can always use the move undo command, even after attacking. That's a bug


- Creating a new squad breaks the threat map. Fix this

- Need to be able to toggle threat layer visualzation to player faction.

- Enemy squads appear to attack when they're out of range


________________________________________________



# Cleanup


- Need a better job of highlighting was data comes from JSON files

- The way the GUI handles keys to track and keyboard presses need to improve



- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 


- CombatController has a lot of artifacts of the old roguelike prior to the change. Specifically in regards to throwables. Find a way tohandle that. 


- Input Package has a lot of leftovers from the old roguelike

- combatqueries.go still has functions which search the entire ecs space



# Bug Fixes

- Throwables are completely broken. There is an out of bounds error


# Review

## The following packages need review

- Guioverworld
- Commander
- GUISquads
- GUIOverworld

____

- The Inventory is a leftover prior to shifting the game to a turn based squad tactics game from a regular roguelike. Think of what you want to do with the inventory

- Look at all of the const. See which ones should be defined in a file





# Additions 


- Add Abilities. Once they are, see how they are called in combat. Detemrine how you want to call them
- Better Combat Log
- Add squad equipment




# Redundancies?

- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead
- Check on ENABLE_COMBAT_LOG and ENABLE_COMBAT_LOG_EXPORT. Probabably some duplication we can get rid of