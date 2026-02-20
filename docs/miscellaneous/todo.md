__________________________________________
#
# # # **IMPORTANT** # # # 


# Raid Package

- Either define archetypes.go in a json file, or use the encounter system. Consider creating a variation of the encounter system which weighs certain units more
    * I.E, "Ambush" - fast units have a weight, ranged battery prioritizes ranged units, etc. 


- Need better reward system. Both raids and regular combat should have rewards. We want rewards to be mostly the same, with additional weighing applies so that raids give better rewards


- Explore the go fix command





----


- Start Random Encounter alwys starts the same encounter

- Identify common patterns in teh different combat start paths. Garrison, raids, and overworld encounters follow different paths.



- Encounters seem to be too skewed towards ranged threats. Using the threat layer visualization as baseline. Rectify that. 

- There has to be a tradeoff between casting spells and moving squads/engaging in combat. Doing both will feel too mechanical. 

- type EventType int in Overworld types is growing too large


 - System for gaining resources. Whether it's through battle or territory tbd


- Overworld node spawning is far too clustered together. All nodes spawn close to eachother. 




- Lots of Empty Claude relate directories. See if there are any issues when removing them

- Consider using an interface for intent in overworld


-  ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  

- Re-neable victory conditions in tickmanager.go. They are disabled for testing

- encounter service should not have mode coordinator. It overlaps too much with what the GUI is doing


- calculateEntryIndexAtPosition in squadeditor_movesquad.go is not accurately calculating the position to move the squad to. Check if there is an ebitenui var (such as entryselected) you can use instead of calculating with pixel positions


- Handle Entity Cleanup

- JSON file error handling


- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer

- Can always use the move undo command, even after attacking. That's a bug




________________________________________________



# Cleanup

- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 


- combatqueries.go still has functions which search the entire ecs space



# Bug Fixes

- Throwables are completely broken. There is an out of bounds error


# Review

## The following packages need review

- Guioverworld
- Commander
- GUISquads
- GUIOverworld
- Spells
- Worldmap

____

x
- Look at all of the const. See which ones should be defined in a file





# Additions 


- Better Combat Log





# Redundancies?

- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead
- Check on ENABLE_COMBAT_LOG and ENABLE_COMBAT_LOG_EXPORT. Probabably some duplication we can get rid of